import SwiftUI

struct TaskSelectionView: View {
    @EnvironmentObject var store: TaskStore
    var onSelect: (String, String, Bool) -> Void   // (taskName, projectSlug, completingPrevious)
    var onCancel: () -> Void
    var completingPrevious: Bool

    @State private var selectedProject: String = "personal"
    @State private var visibleRecentTasks: [TaskStore.RecentTask] = []
    @State private var newTaskText: String = ""
    @FocusState private var fieldFocused: Bool

    private let formatter: RelativeDateTimeFormatter = {
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .abbreviated
        return f
    }()

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Select task")
                    .font(.headline)
                Spacer()
                Button("Cancel", action: onCancel)
                    .buttonStyle(.plain)
                    .foregroundColor(.secondary)
                    .font(.callout)
            }
            .padding(.horizontal, 12)
            .padding(.top, 12)
            .padding(.bottom, 8)

            Divider()

            HStack(spacing: 6) {
                Text("Project")
                    .font(.caption)
                    .foregroundColor(.secondary)
                Spacer()
                Picker("", selection: $selectedProject) {
                    ForEach(store.availableProjects, id: \.self) { project in
                        Text(project).tag(project)
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
                .frame(maxWidth: 160)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 6)
            .onChange(of: selectedProject) { _ in
                refreshVisibleRecentTasks()
            }

            Divider()

            if visibleRecentTasks.isEmpty {
                Text("No recent tasks")
                    .font(.callout)
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity, alignment: .center)
                    .padding(.vertical, 16)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(visibleRecentTasks) { task in
                            Button {
                                onSelect(task.name, selectedProject, completingPrevious)
                            } label: {
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(task.name)
                                            .font(.callout)
                                            .foregroundColor(.primary)
                                            .lineLimit(1)
                                        Text(formatter.localizedString(for: task.startedAt, relativeTo: Date()))
                                            .font(.caption2)
                                            .foregroundColor(.secondary)
                                    }
                                    Spacer()
                                }
                                .padding(.horizontal, 12)
                                .padding(.vertical, 7)
                                .contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .frame(maxHeight: 180)

                Divider()
            }

            HStack(spacing: 6) {
                TextField("New task…", text: $newTaskText)
                    .textFieldStyle(.plain)
                    .font(.callout)
                    .focused($fieldFocused)
                    .onSubmit { commitNewTask() }

                Button("Start", action: commitNewTask)
                    .buttonStyle(.borderedProminent)
                    .controlSize(.small)
                    .disabled(newTaskText.trimmingCharacters(in: .whitespaces).isEmpty)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        .frame(width: 280)
        .onAppear {
            selectedProject = store.currentProjectSlug
            refreshVisibleRecentTasks()
            // Deferred a tick: the project Picker's onChange-driven state
            // mutation above can still be settling into a re-render at the
            // moment onAppear fires, and requesting focus mid-churn is a
            // known way for @FocusState to silently lose the request in an
            // NSPopover. Letting that settle first is what actually gets the
            // field focused reliably.
            DispatchQueue.main.async {
                fieldFocused = true
            }
        }
    }

    private func refreshVisibleRecentTasks() {
        visibleRecentTasks = store.recentTasks(forProject: selectedProject)
    }

    private func commitNewTask() {
        let name = newTaskText.trimmingCharacters(in: .whitespaces)
        guard !name.isEmpty else { return }
        onSelect(name, selectedProject, completingPrevious)
    }
}
