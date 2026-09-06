import Foundation

struct TaskRow {
    let uuid: String
    let task: String
    let from: Date
    let to: Date?
    let completed: Bool?
}

struct CSVLogger {

    // MARK: - Data directory

    /// The Go CLI's own config file — internal/config.go resolves this exact
    /// path via os.UserConfigDir(), which is ~/Library/Application Support on
    /// macOS. The widget and CLI deliberately share this one file rather than
    /// each keeping their own copy of "which directory" (the widget used to
    /// use UserDefaults) — one file, read/written by both sides, means
    /// there's no separate copy to fall out of sync in the first place.
    private static var cliConfigURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Application Support/focuson/config.toml")
    }

    /// Reads `data_dir = "..."` out of the CLI's config.toml. Deliberately
    /// not a general TOML parser (matches the "zero TOML parsing" rule the
    /// project picker follows too) — just enough to find one flat key, and
    /// it tolerates the CLI adding other keys around it later since it scans
    /// line by line rather than assuming file structure.
    private static func readCLIConfigDataDir() -> String? {
        guard let contents = try? String(contentsOf: cliConfigURL, encoding: .utf8) else { return nil }
        for line in contents.split(separator: "\n") {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard trimmed.hasPrefix("data_dir") else { continue }
            guard let eq = trimmed.firstIndex(of: "=") else { continue }
            var value = trimmed[trimmed.index(after: eq)...].trimmingCharacters(in: .whitespaces)
            if value.hasPrefix("\""), value.hasSuffix("\""), value.count >= 2 {
                value = String(value.dropFirst().dropLast())
                value = value.replacingOccurrences(of: "\\\"", with: "\"").replacingOccurrences(of: "\\\\", with: "\\")
            }
            return value
        }
        return nil
    }

    static var dataDirectoryURL: URL {
        if let stored = readCLIConfigDataDir(), !stored.isEmpty {
            return URL(fileURLWithPath: stored)
        }
        return FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("focuson-data")
    }

    static var dataDirectoryDisplayPath: String {
        let path = dataDirectoryURL.path
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        return path.hasPrefix(home) ? "~" + path.dropFirst(home.count) : path
    }

    static func setDataDirectory(_ url: URL) {
        writeCLIConfig(dataDir: url.path)
    }

    private static func writeCLIConfig(dataDir: String) {
        let escaped = dataDir.replacingOccurrences(of: "\\", with: "\\\\").replacingOccurrences(of: "\"", with: "\\\"")
        let contents = "data_dir = \"\(escaped)\"\n"
        let url = cliConfigURL
        try? FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try? contents.write(to: url, atomically: true, encoding: .utf8)
    }

    private static var projectsDirectoryURL: URL {
        dataDirectoryURL.appendingPathComponent("projects")
    }

    /// The widget's only source of truth for "which projects exist" — it
    /// never parses manifest.toml (that's the CLI's file). See spec_v2.md,
    /// Widget Changes → "Project picker".
    static func listProjectSlugs() -> [String] {
        let fm = FileManager.default
        guard let entries = try? fm.contentsOfDirectory(
            at: projectsDirectoryURL, includingPropertiesForKeys: [.isDirectoryKey], options: [.skipsHiddenFiles]
        ) else {
            return []
        }
        return entries.compactMap { url -> String? in
            var isDir: ObjCBool = false
            guard fm.fileExists(atPath: url.path, isDirectory: &isDir), isDir.boolValue else { return nil }
            return url.lastPathComponent
        }.sorted()
    }

    /// Creates the data directory and, if nothing is there yet, a minimal
    /// manifest.toml (just the "personal" project) plus projects/personal/.
    /// Mirrors the CLI's own bootstrap (internal/manifest.Bootstrap) so
    /// either side can be first to touch a brand-new data directory. Never
    /// overwrites an existing manifest — the widget never writes
    /// manifest.toml otherwise.
    static func bootstrapDataDirectoryIfNeeded() {
        let fm = FileManager.default

        // If neither side has ever set a directory, dataDirectoryURL is
        // just an implicit fallback (~/focuson-data), not yet a real,
        // recorded choice. Persist it explicitly so a CLI run afterward
        // sees the same path instead of running its own first-run setup
        // and potentially landing somewhere else.
        if !fm.fileExists(atPath: cliConfigURL.path) {
            writeCLIConfig(dataDir: dataDirectoryURL.path)
        }

        try? fm.createDirectory(at: dataDirectoryURL, withIntermediateDirectories: true)

        let manifestURL = dataDirectoryURL.appendingPathComponent("manifest.toml")
        if !fm.fileExists(atPath: manifestURL.path) {
            let minimalManifest = """
            [business]
            name = ""
            address = ""
            email = ""
            payment_details = ""

            [[project]]
            slug = "personal"
            client = ""
            name = "Personal"

            """
            try? minimalManifest.write(to: manifestURL, atomically: true, encoding: .utf8)
        }

        try? fm.createDirectory(
            at: projectsDirectoryURL.appendingPathComponent("personal"),
            withIntermediateDirectories: true
        )
    }

    // MARK: - Per-project task_log.csv

    static func fileURL(project: String) -> URL {
        projectsDirectoryURL.appendingPathComponent(project).appendingPathComponent("task_log.csv")
    }

    private static let iso8601: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime]
        return f
    }()

    static func createFileIfNeeded(project: String) {
        let url = fileURL(project: project)
        guard !FileManager.default.fileExists(atPath: url.path) else { return }
        try? FileManager.default.createDirectory(at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        let header = "uuid,task,from,to,completed\n"
        try? header.write(to: url, atomically: true, encoding: .utf8)
    }

    static func appendRow(project: String, uuid: String, task: String, from: Date, to: Date?, completed: Bool?) {
        createFileIfNeeded(project: project)
        let url = fileURL(project: project)
        let escapedTask = task.replacingOccurrences(of: "\"", with: "\"\"")
        let fromStr = iso8601.string(from: from)
        let toStr = to.map { iso8601.string(from: $0) } ?? ""
        let completedStr: String
        if let c = completed {
            completedStr = c ? "true" : "false"
        } else {
            completedStr = ""
        }
        let line = "\(uuid),\"\(escapedTask)\",\(fromStr),\(toStr),\(completedStr)\n"
        guard let data = line.data(using: .utf8) else { return }
        if let handle = try? FileHandle(forWritingTo: url) {
            handle.seekToEndOfFile()
            handle.write(data)
            try? handle.close()
        } else {
            try? data.write(to: url)
        }
    }

    static func readAllRows(project: String) -> [TaskRow] {
        let url = fileURL(project: project)
        guard let content = try? String(contentsOf: url, encoding: .utf8) else { return [] }
        var lines = content.components(separatedBy: "\n")
        guard lines.count > 1 else { return [] }
        lines.removeFirst() // header
        var rows: [TaskRow] = []
        for line in lines {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard !trimmed.isEmpty else { continue }
            if let row = parseRow(trimmed) {
                rows.append(row)
            }
        }
        return rows
    }

    private static func parseRow(_ line: String) -> TaskRow? {
        var fields: [String] = []
        var current = ""
        var inQuotes = false
        var i = line.startIndex
        while i < line.endIndex {
            let c = line[i]
            if inQuotes {
                if c == "\"" {
                    let next = line.index(after: i)
                    if next < line.endIndex && line[next] == "\"" {
                        current.append("\"")
                        i = line.index(after: next)
                        continue
                    } else {
                        inQuotes = false
                    }
                } else {
                    current.append(c)
                }
            } else {
                if c == "\"" {
                    inQuotes = true
                } else if c == "," {
                    fields.append(current)
                    current = ""
                } else {
                    current.append(c)
                }
            }
            i = line.index(after: i)
        }
        fields.append(current)
        guard fields.count >= 5 else { return nil }
        let uuid = fields[0]
        let taskName = fields[1]
        guard let fromDate = iso8601.date(from: fields[2]) else { return nil }
        let toDate = fields[3].isEmpty ? nil : iso8601.date(from: fields[3])
        let completed: Bool? = fields[4].isEmpty ? nil : (fields[4] == "true")
        return TaskRow(uuid: uuid, task: taskName, from: fromDate, to: toDate, completed: completed)
    }
}
