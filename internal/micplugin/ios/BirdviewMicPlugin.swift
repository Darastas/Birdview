import AVFoundation
import Foundation

final class BirdviewMicPlugin: NSObject, MyGoPlugin {
    private var recorder: AVAudioRecorder?
    private var tempURL: URL?
    private var meterTimer: Timer?
    private var startedAt: Date?

    func call(_ method: String, _ args: [String: Any], _ reply: @escaping PluginReply) {
        switch method {
        case "start":
            start(reply)
        case "stop":
            stop(args, reply)
        case "cancel":
            cancel(reply)
        default:
            reply(.failure(PluginFailure("NoMethod", method)))
        }
    }

    private func start(_ reply: @escaping PluginReply) {
        if recorder?.isRecording == true {
            reply(.failure(PluginFailure("Busy", "当前正在录音中")))
            return
        }
        let session = AVAudioSession.sharedInstance()
        session.requestRecordPermission { [weak self] granted in
            DispatchQueue.main.async {
                guard let self else { return }
                guard granted else {
                    reply(.failure(PluginFailure("Denied", "未授予麦克风录音权限，请在系统设置中开启或使用导入音频")))
                    return
                }
                do {
                    try session.setCategory(.playAndRecord, mode: .default, options: [.defaultToSpeaker])
                    try session.setActive(true)
                    let tmp = FileManager.default.temporaryDirectory.appendingPathComponent("mic_\(Int64(Date().timeIntervalSince1970 * 1000)).wav")
                    let settings: [String: Any] = [
                        AVFormatIDKey: kAudioFormatLinearPCM,
                        AVSampleRateKey: 44100.0,
                        AVNumberOfChannelsKey: 1,
                        AVLinearPCMBitDepthKey: 16,
                        AVLinearPCMIsFloatKey: false,
                        AVLinearPCMIsBigEndianKey: false,
                    ]
                    let rec = try AVAudioRecorder(url: tmp, settings: settings)
                    rec.isMeteringEnabled = true
                    guard rec.record() else {
                        reply(.failure(PluginFailure("Failed", "无法开始麦克风录音")))
                        return
                    }
                    self.recorder = rec
                    self.tempURL = tmp
                    self.startedAt = Date()
                    self.meterTimer?.invalidate()
                    self.meterTimer = Timer.scheduledTimer(withTimeInterval: 0.08, repeats: true) { [weak self] _ in
                        guard let self, let r = self.recorder, r.isRecording else { return }
                        r.updateMeters()
                        let db = Double(r.averagePower(forChannel: 0))
                        let level = max(0.0, min(1.0, (db + 55.0) / 55.0))
                        let elapsedMs = Int64((Date().timeIntervalSince(self.startedAt ?? Date())) * 1000)
                        PluginHost.emit("birdview_mic", "level", ["level": level, "elapsedMs": elapsedMs])
                    }
                    reply(.success(["started": true]))
                } catch {
                    reply(.failure(PluginFailure("Failed", error.localizedDescription)))
                }
            }
        }
    }

    private func stop(_ args: [String: Any], _ reply: @escaping PluginReply) {
        meterTimer?.invalidate()
        meterTimer = nil
        guard let rec = recorder, let src = tempURL else {
            reply(.failure(PluginFailure("Failed", "当前未在录音")))
            return
        }
        let elapsedMs = Int64((Date().timeIntervalSince(startedAt ?? Date())) * 1000)
        rec.stop()
        recorder = nil
        tempURL = nil
        try? AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)

        let targetPath = (args["path"] as? String)?.trimmingCharacters(in: .whitespacesAndNewlines) ?? ""
        if targetPath.isEmpty {
            reply(.success(["path": src.path, "elapsedMs": elapsedMs]))
            return
        }
        let dst = URL(fileURLWithPath: targetPath)
        do {
            try FileManager.default.createDirectory(at: dst.deletingLastPathComponent(), withIntermediateDirectories: true)
            if FileManager.default.fileExists(atPath: dst.path) {
                try FileManager.default.removeItem(at: dst)
            }
            try FileManager.default.moveItem(at: src, to: dst)
            reply(.success(["path": dst.path, "elapsedMs": elapsedMs]))
        } catch {
            reply(.failure(PluginFailure("Failed", error.localizedDescription)))
        }
    }

    private func cancel(_ reply: @escaping PluginReply) {
        meterTimer?.invalidate()
        meterTimer = nil
        recorder?.stop()
        recorder = nil
        if let src = tempURL {
            try? FileManager.default.removeItem(at: src)
        }
        tempURL = nil
        try? AVAudioSession.sharedInstance().setActive(false, options: .notifyOthersOnDeactivation)
        reply(.success(nil))
    }
}
