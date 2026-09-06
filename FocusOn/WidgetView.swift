import SwiftUI
import AppKit

struct WidgetView: View {
    @EnvironmentObject var store: TaskStore

    @State private var pulseOpacity: Double = 1.0
    @State private var now: Date = Date()
    private static let activeBlue = Color(red: 0.4, green: 0.68, blue: 1.0)

    var body: some View {
        ZStack {
            RoundedRectangle(cornerRadius: 12)
                .fill(Color(nsColor: .windowBackgroundColor))
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(Color(nsColor: .separatorColor), lineWidth: 1)
                )
                .shadow(color: .black.opacity(0.18), radius: 8, x: 0, y: 3)

            HStack(spacing: 8) {
                let isActive = store.currentTaskName != nil
                Circle()
                    .fill(isActive ? Self.activeBlue : Color(nsColor: .tertiaryLabelColor))
                    .frame(width: 24, height: 24)
                    .opacity(pulseOpacity)
                    .onAppear { updatePulse(active: isActive) }
                    .onChange(of: store.currentTaskName) { _ in
                        updatePulse(active: store.currentTaskName != nil)
                    }

                if let task = store.currentTaskName {
                    Text(task)
                        .font(.system(size: 15, design: .rounded))
                        .foregroundColor(.primary)
                        .lineLimit(1)
                        .fixedSize()
                    if let start = store.currentTaskStartedAt {
                        Text(elapsedString(from: start, to: now))
                            .font(.system(size: 13, design: .monospaced))
                            .foregroundColor(.secondary)
                            .fixedSize()
                    }
                } else {
                    Text("No active task")
                        .font(.system(size: 15, design: .rounded))
                        .foregroundColor(.secondary)
                        .lineLimit(1)
                        .fixedSize()
                }

                Image(systemName: "chevron.down")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 10)
        }
        // No SwiftUI gestures — drag/tap handled by WidgetContainerView at the NSView level
        .onReceive(Timer.publish(every: 1, on: .main, in: .common).autoconnect()) { tick in
            now = tick
        }
    }

    private func elapsedString(from start: Date, to end: Date) -> String {
        let total = max(0, Int(end.timeIntervalSince(start)))
        let h = total / 3600
        let m = (total % 3600) / 60
        let s = total % 60
        if h > 0 {
            return String(format: "%d:%02d:%02d", h, m, s)
        } else {
            return String(format: "%d:%02d", m, s)
        }
    }

    private func updatePulse(active: Bool) {
        if active {
            pulseOpacity = 1.0
            withAnimation(.easeInOut(duration: 1).repeatForever(autoreverses: true)) {
                pulseOpacity = 0.4
            }
        } else {
            withAnimation(.linear(duration: 0)) {
                pulseOpacity = 1.0
            }
        }
    }
}
