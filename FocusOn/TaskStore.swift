import Foundation
import Combine

@MainActor
final class TaskStore: ObservableObject {
    @Published var currentTaskName: String? = nil
    @Published var currentTaskStartedAt: Date? = nil

    /// The UUID shared by the active task's opening row and whatever row
    /// eventually closes it (see CSVLogger.appendRow). Generated fresh each
    /// time a task starts; never reused across sessions.
    @Published var currentTaskUUID: String? = nil

    /// The project the *currently active* task belongs to. Persists even
    /// after the task completes/pauses, so it doubles as "last used project"
    /// — the task-selection picker's default next time it opens.
    @Published var currentProjectSlug: String = "personal"

    /// Source of truth is CSVLogger.listProjectSlugs() (directory listing of
    /// projects/), never manifest.toml — see CSVLogger's doc comment.
    @Published var availableProjects: [String] = ["personal"]

    struct RecentTask: Identifiable {
        let id = UUID()
        let name: String
        let startedAt: Date
    }

    func loadState() {
        let defaults = UserDefaults.standard
        currentTaskName = defaults.string(forKey: "focuson.currentTaskName")
        currentProjectSlug = defaults.string(forKey: "focuson.currentProjectSlug") ?? "personal"
        CSVLogger.bootstrapDataDirectoryIfNeeded()
        refreshAvailableProjects()

        guard let name = currentTaskName else { return }

        // Whatever was active at last quit already got a closing row written
        // by writeClosingRowOnTerminate (graceful quit) — or, on a crash, was
        // never closed and stays a permanently orphaned open row (documented
        // v1 limitation). Either way the old uuid's story is already told on
        // disk. "Resuming" the same task here is therefore a brand new
        // session — new uuid, new start time — never a reuse of the old
        // uuid, which would otherwise produce two `to`-filled rows sharing
        // one uuid and break the one-row-per-uuid invariant invoicing
        // depends on.
        let now = Date()
        let uuid = UUID().uuidString.lowercased()
        currentTaskStartedAt = now
        currentTaskUUID = uuid
        persistState()
        CSVLogger.appendRow(project: currentProjectSlug, uuid: uuid, task: name, from: now, to: nil, completed: nil)
    }

    func refreshAvailableProjects() {
        let projects = CSVLogger.listProjectSlugs()
        availableProjects = projects.isEmpty ? ["personal"] : projects
    }

    /// Recent incomplete tasks scoped to one project — task names aren't
    /// deduplicated across projects since they live in separate CSV files.
    /// Called directly by the task-selection UI as its picked project
    /// changes, rather than cached on the store.
    func recentTasks(forProject project: String) -> [RecentTask] {
        let rows = CSVLogger.readAllRows(project: project)
        // Include open rows (no closing time) and paused rows (closed but not completed).
        // Exclude rows explicitly marked completed = true.
        let incomplete = rows.filter { $0.completed != true }
        var seen: [String: Date] = [:]
        for row in incomplete.reversed() {
            if seen[row.task] == nil {
                seen[row.task] = row.from
            }
        }
        return seen
            .map { RecentTask(name: $0.key, startedAt: $0.value) }
            .sorted { $0.startedAt > $1.startedAt }
            .prefix(10)
            .map { $0 }
    }

    func startTask(_ name: String, project: String, completingPrevious: Bool) {
        let now = Date()
        if let prev = currentTaskName, let prevStart = currentTaskStartedAt, let prevUUID = currentTaskUUID {
            CSVLogger.appendRow(project: currentProjectSlug, uuid: prevUUID, task: prev, from: prevStart, to: now, completed: completingPrevious ? true : false)
        }
        let newUUID = UUID().uuidString.lowercased()
        currentTaskName = name
        currentProjectSlug = project
        currentTaskStartedAt = now
        currentTaskUUID = newUUID
        persistState()
        CSVLogger.appendRow(project: project, uuid: newUUID, task: name, from: now, to: nil, completed: nil)
    }

    func completeCurrentTask() {
        guard let name = currentTaskName, let start = currentTaskStartedAt, let uuid = currentTaskUUID else { return }
        let now = Date()
        CSVLogger.appendRow(project: currentProjectSlug, uuid: uuid, task: name, from: start, to: now, completed: true)
        currentTaskName = nil
        currentTaskStartedAt = nil
        currentTaskUUID = nil
        persistState()
    }

    func pauseCurrentTask() {
        guard let name = currentTaskName, let start = currentTaskStartedAt, let uuid = currentTaskUUID else { return }
        CSVLogger.appendRow(project: currentProjectSlug, uuid: uuid, task: name, from: start, to: Date(), completed: false)
        currentTaskName = nil
        currentTaskStartedAt = nil
        currentTaskUUID = nil
        persistState()
    }

    /// Writes a single fully-formed row directly — for time you forgot to
    /// start the timer for. No pairing (both from and to are already known),
    /// and no interaction with currentTaskName/currentTaskStartedAt/
    /// currentTaskUUID: this never touches whatever task is currently being
    /// tracked live.
    func logPastSession(project: String, task: String, from: Date, to: Date, completed: Bool) {
        let uuid = UUID().uuidString.lowercased()
        CSVLogger.appendRow(project: project, uuid: uuid, task: task, from: from, to: to, completed: completed)
    }

    func writeClosingRowOnTerminate() {
        guard let name = currentTaskName, let start = currentTaskStartedAt, let uuid = currentTaskUUID else { return }
        CSVLogger.appendRow(project: currentProjectSlug, uuid: uuid, task: name, from: start, to: Date(), completed: false)
    }

    private func persistState() {
        let defaults = UserDefaults.standard
        defaults.set(currentTaskName, forKey: "focuson.currentTaskName")
        defaults.set(currentProjectSlug, forKey: "focuson.currentProjectSlug")
        defaults.set(currentTaskUUID, forKey: "focuson.currentTaskUUID")
        if let date = currentTaskStartedAt {
            defaults.set(date.timeIntervalSince1970, forKey: "focuson.currentTaskStartedAt")
        } else {
            defaults.removeObject(forKey: "focuson.currentTaskStartedAt")
        }
    }
}
