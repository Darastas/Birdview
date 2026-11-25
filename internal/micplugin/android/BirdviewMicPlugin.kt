package dev.mygo

import android.Manifest
import android.annotation.SuppressLint
import android.media.AudioFormat
import android.media.AudioRecord
import android.media.MediaRecorder
import android.os.SystemClock
import org.json.JSONObject
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream
import kotlin.math.sqrt

class BirdviewMicPlugin : MyGoPlugin {
    private val lock = Any()
    @Volatile private var recording = false
    private var audioRecord: AudioRecord? = null
    private var worker: Thread? = null
    private var pcmFile: File? = null
    private var startedAtMs: Long = 0

    override fun call(method: String, args: JSONObject, reply: PluginReply) {
        when (method) {
            "start" -> start(reply)
            "stop" -> stop(args, reply)
            "cancel" -> cancel(reply)
            else -> reply.failure(PluginFailure("NoMethod", method))
        }
    }

    private fun start(reply: PluginReply) {
        if (recording) {
            reply.failure(PluginFailure("Busy", "当前正在录音中"))
            return
        }
        PluginHost.requestPermissions(arrayOf(Manifest.permission.RECORD_AUDIO)) { granted ->
            if (!granted) {
                reply.failure(PluginFailure("Denied", "未授予麦克风录音权限，请在系统设置中开启或使用导入音频"))
                return@requestPermissions
            }
            beginCapture(reply)
        }
    }

    @SuppressLint("MissingPermission")
    private fun beginCapture(reply: PluginReply) {
        synchronized(lock) {
            if (recording) {
                reply.failure(PluginFailure("Busy", "当前正在录音中"))
                return
            }
            val sampleRate = 44100
            val channelCfg = AudioFormat.CHANNEL_IN_MONO
            val audioFmt = AudioFormat.ENCODING_PCM_16BIT
            val minBuf = AudioRecord.getMinBufferSize(sampleRate, channelCfg, audioFmt)
            if (minBuf <= 0) {
                reply.failure(PluginFailure("Failed", "设备不支持 44.1kHz 单声道麦克风采集"))
                return
            }
            val bufSize = maxOf(minBuf * 2, sampleRate * 2 / 5)
            val ar = try {
                AudioRecord(MediaRecorder.AudioSource.MIC, sampleRate, channelCfg, audioFmt, bufSize)
            } catch (e: Exception) {
                reply.failure(PluginFailure("Failed", "无法初始化麦克风: ${e.message}"))
                return
            }
            if (ar.state != AudioRecord.STATE_INITIALIZED) {
                ar.release()
                reply.failure(PluginFailure("Failed", "麦克风硬件初始化失败"))
                return
            }

            val raw = File(PluginHost.context.cacheDir, "mic_raw_${System.nanoTime()}.pcm")
            raw.parentFile?.mkdirs()

            try {
                ar.startRecording()
            } catch (e: Exception) {
                ar.release()
                reply.failure(PluginFailure("Failed", "启动麦克风采集失败: ${e.message}"))
                return
            }

            audioRecord = ar
            pcmFile = raw
            recording = true
            startedAtMs = SystemClock.elapsedRealtime()

            val readChunk = ByteArray(sampleRate * 2 * 80 / 1000) // ~80ms
            val t = Thread({
                var fos: FileOutputStream? = null
                try {
                    fos = FileOutputStream(raw)
                    while (recording) {
                        val n = ar.read(readChunk, 0, readChunk.size)
                        if (n > 0) {
                            fos.write(readChunk, 0, n)
                            val level = calcPCM16Level(readChunk, n)
                            val elapsed = SystemClock.elapsedRealtime() - startedAtMs
                            PluginHost.emit(
                                "birdview_mic",
                                "level",
                                JSONObject().put("level", level).put("elapsedMs", elapsed)
                            )
                        }
                    }
                    fos.flush()
                } catch (_: Exception) {
                } finally {
                    try { fos?.close() } catch (_: Exception) {}
                }
            }, "BirdviewMicCapture")
            worker = t
            t.start()
            reply.success(JSONObject().put("started", true))
        }
    }

    private fun stopInternal(): Pair<File?, Long> {
        val t: Thread?
        val ar: AudioRecord?
        val raw: File?
        val elapsed: Long
        synchronized(lock) {
            if (!recording && audioRecord == null) {
                return Pair(null, 0L)
            }
            recording = false
            elapsed = maxOf(0L, SystemClock.elapsedRealtime() - startedAtMs)
            ar = audioRecord
            t = worker
            raw = pcmFile
            audioRecord = null
            worker = null
            pcmFile = null
        }
        try { ar?.stop() } catch (_: Exception) {}
        try { t?.join(1500) } catch (_: Exception) {}
        try { ar?.release() } catch (_: Exception) {}
        return Pair(raw, elapsed)
    }

    private fun stop(args: JSONObject, reply: PluginReply) {
        val targetPath = args.optString("path", "").trim()
        Thread({
            val (raw, elapsed) = stopInternal()
            if (raw == null || !raw.exists() || raw.length() <= 0) {
                raw?.delete()
                reply.failure(PluginFailure("Failed", "未采集到有效的音频数据"))
                return@Thread
            }
            val dest = if (targetPath.isNotEmpty()) {
                File(targetPath)
            } else {
                val dir = File(PluginHost.context.cacheDir, "mic").apply { mkdirs() }
                File(dir, "rec_${System.currentTimeMillis()}.wav")
            }
            try {
                dest.parentFile?.mkdirs()
                writeWavFile(raw, dest, 44100, 1, 16)
                raw.delete()
                reply.success(JSONObject().put("path", dest.absolutePath).put("elapsedMs", elapsed))
            } catch (e: Exception) {
                raw.delete()
                reply.failure(PluginFailure("Failed", "保存 WAV 录音失败: ${e.message}"))
            }
        }, "BirdviewMicStop").start()
    }

    private fun cancel(reply: PluginReply) {
        Thread({
            val (raw, _) = stopInternal()
            raw?.delete()
            reply.success(null)
        }, "BirdviewMicCancel").start()
    }

    private fun calcPCM16Level(buf: ByteArray, len: Int): Double {
        val samples = len / 2
        if (samples <= 0) return 0.0
        var sumSq = 0.0
        for (i in 0 until samples) {
            val lo = buf[i * 2].toInt() and 0xff
            val hi = buf[i * 2 + 1].toInt()
            val sample = (hi shl 8) or lo
            val norm = sample / 32768.0
            sumSq += norm * norm
        }
        val rms = sqrt(sumSq / samples)
        return (rms * 3.5).coerceIn(0.0, 1.0)
    }

    private fun writeWavFile(pcmFile: File, dest: File, sampleRate: Int, channels: Int, bitsPerSample: Int) {
        val pcmLen = pcmFile.length().toInt()
        val byteRate = sampleRate * channels * bitsPerSample / 8
        val blockAlign = channels * bitsPerSample / 8
        val header = ByteArray(44)
        header[0] = 'R'.code.toByte(); header[1] = 'I'.code.toByte(); header[2] = 'F'.code.toByte(); header[3] = 'F'.code.toByte()
        putLEInt(header, 4, 36 + pcmLen)
        header[8] = 'W'.code.toByte(); header[9] = 'A'.code.toByte(); header[10] = 'V'.code.toByte(); header[11] = 'E'.code.toByte()
        header[12] = 'f'.code.toByte(); header[13] = 'm'.code.toByte(); header[14] = 't'.code.toByte(); header[15] = ' '.code.toByte()
        putLEInt(header, 16, 16)
        putLEShort(header, 20, 1)
        putLEShort(header, 22, channels)
        putLEInt(header, 24, sampleRate)
        putLEInt(header, 28, byteRate)
        putLEShort(header, 32, blockAlign)
        putLEShort(header, 34, bitsPerSample)
        header[36] = 'd'.code.toByte(); header[37] = 'a'.code.toByte(); header[38] = 't'.code.toByte(); header[39] = 'a'.code.toByte()
        putLEInt(header, 40, pcmLen)

        FileOutputStream(dest).use { out ->
            out.write(header)
            FileInputStream(pcmFile).use { input ->
                val copyBuf = ByteArray(16384)
                while (true) {
                    val n = input.read(copyBuf)
                    if (n <= 0) break
                    out.write(copyBuf, 0, n)
                }
            }
            out.flush()
        }
    }

    private fun putLEInt(dst: ByteArray, off: Int, v: Int) {
        dst[off] = (v and 0xff).toByte()
        dst[off + 1] = ((v ushr 8) and 0xff).toByte()
        dst[off + 2] = ((v ushr 16) and 0xff).toByte()
        dst[off + 3] = ((v ushr 24) and 0xff).toByte()
    }

    private fun putLEShort(dst: ByteArray, off: Int, v: Int) {
        dst[off] = (v and 0xff).toByte()
        dst[off + 1] = ((v ushr 8) and 0xff).toByte()
    }
}
