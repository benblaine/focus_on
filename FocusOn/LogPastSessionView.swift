import SwiftUI

/// For time you forgot to start the timer for. Writes a single fully-formed
/// row directly (see TaskStore.logPastSession) — independent of whatever
/// task (if any) is currently being tracked live.
struct LogPastSessionView: View {
    @EnvironmentObject var store: TaskStore
    var onSave: (String, String, Date, Date, Bool) -> Void  // (project, task, from, to, completed)
    var onCancel: () -> Void

    @State private var selectedProject: String = "personal"
    @State private var taskText: String = ""
    @State private var from: Date = Date().addingTimeInterval(-3600)
    @State private var to: Date = Date()
    @State private var completed: Bool = true
    @FocusState private var taskFieldFocused: Bool

    private var isValid: Bool {
        !taskText.trimmingCharacters(in: .whitespaces).isEmpty && to > from
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text("Log past session")
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

            VStack(alignment: .leading, spacing: 10) {
                HStack {
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

                TextField("Task…", text: $taskText)
                    .textFieldStyle(.roundedBorder)
                    .font(.callout)
                    .focused($taskFieldFocused)

                DatePicker("From", selection: $from)
                    .font(.caption)

                DatePicker("To", selection: $to)
                    .font(.caption)

                if to <= from {
                    Text("To must be after From")
                        .font(.caption2)
                        .foregroundColor(.red)
                }

                Toggle("Completed", isOn: $completed)
                    .toggleStyle(.checkbox)
                    .font(.caption)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)

            Divider()

            HStack {
                Spacer()
                Button("Save") {
                    onSave(selectedProject, taskText.trimmingCharacters(in: .whitespaces), from, to, completed)
                }
                .buttonStyle(.borderedProminent)
                .controlSize(.small)
                .disabled(!isValid)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        .frame(width: 300)
        .onAppear {
            selectedProject = store.currentProjectSlug
            taskFieldFocused = true
        }
    }
}
