package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// VoiceActivityDetector handles real-time voice activity detection
type VoiceActivityDetector struct {
	audioFile          string
	silenceThreshold   float64
	minSpeechDuration  time.Duration
	maxSilenceDuration time.Duration
}

// NewVoiceActivityDetector creates a new VAD instance
func NewVoiceActivityDetector(audioFile string) *VoiceActivityDetector {
	return &VoiceActivityDetector{
		audioFile:          audioFile,
		silenceThreshold:   SilenceThreshold,
		minSpeechDuration:  time.Duration(MinSpeechDuration) * time.Second,
		maxSilenceDuration: time.Duration(MaxSilenceDuration) * time.Second,
	}
}

// MonitorWithSoX uses SoX for more accurate voice activity detection
func (vad *VoiceActivityDetector) MonitorWithSoX(ctx context.Context, recordCmd *exec.Cmd) {
	defer func() {
		if recordCmd.Process != nil {
			recordCmd.Process.Kill()
		}
	}()

	speechDetected := false
	speechStartTime := time.Time{}
	lastActivityTime := time.Now()

	// Give initial buffer time
	time.Sleep(500 * time.Millisecond)

	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check if the audio file exists and analyze it
			if _, err := os.Stat(vad.audioFile); err == nil {
				amplitude := vad.getAudioAmplitude()

				if amplitude > vad.silenceThreshold {
					// Speech detected
					if !speechDetected {
						speechDetected = true
						speechStartTime = time.Now()
						log.Println("[VAD]: Speech detected")
					}
					lastActivityTime = time.Now()
				} else if speechDetected {
					// Check if we've had enough silence
					silenceDuration := time.Since(lastActivityTime)
					speechDuration := time.Since(speechStartTime)

					if speechDuration >= vad.minSpeechDuration && silenceDuration >= vad.maxSilenceDuration {
						log.Printf("[VAD]: End of speech detected (spoke for %.1fs, silent for %.1fs)", speechDuration.Seconds(), silenceDuration.Seconds())
						return
					}
				}

				// Safety timeout - don't record for more than 30 seconds
				if speechDetected && time.Since(speechStartTime) > 30*time.Second {
					log.Println("[VAD]: Maximum recording time reached")
					return
				}
			}
		}
	}
}

// getAudioAmplitude analyzes the current audio file to get amplitude level
func (vad *VoiceActivityDetector) getAudioAmplitude() float64 {
	// Use SoX to get audio statistics
	if _, err := exec.LookPath("sox"); err == nil {
		return vad.getAmplitudeWithSoX()
	}

	// Fallback: use ffmpeg to get audio level
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return vad.getAmplitudeWithFFmpeg()
	}

	// Basic fallback: check file size changes
	if info, err := os.Stat(vad.audioFile); err == nil {
		return float64(info.Size()) / 100000.0 // Very rough heuristic
	}

	return 0.0
}

// getAmplitudeWithSoX uses SoX to get accurate audio amplitude
func (vad *VoiceActivityDetector) getAmplitudeWithSoX() float64 {
	// Get the last 0.5 seconds of audio for analysis
	cmd := exec.Command("sox", vad.audioFile, "-t", "wav", "-", "trim", "-0.5")

	// Pipe to sox stat to get RMS amplitude
	statCmd := exec.Command("sox", "-t", "wav", "-", "-n", "stat")

	// Connect the commands
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0.0
	}
	statCmd.Stdin = pipe

	output, err := statCmd.StdoutPipe()
	if err != nil {
		pipe.Close()
		return 0.0
	}

	if err := cmd.Start(); err != nil {
		pipe.Close()
		output.Close()
		return 0.0
	}

	if err := statCmd.Start(); err != nil {
		pipe.Close()
		output.Close()
		cmd.Wait()
		return 0.0
	}

	// Read the statistics output
	scanner := bufio.NewScanner(output)
	rmsAmplitude := 0.0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "RMS amplitude") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				if val, err := strconv.ParseFloat(parts[2], 64); err == nil {
					rmsAmplitude = val
					break
				}
			}
		}
	}

	output.Close()
	pipe.Close()
	cmd.Wait()
	statCmd.Wait()

	return rmsAmplitude
}

// getAmplitudeWithFFmpeg uses ffmpeg to get audio amplitude
func (vad *VoiceActivityDetector) getAmplitudeWithFFmpeg() float64 {
	// Use ffmpeg to get volume level of the last part of the file
	cmd := exec.Command("ffmpeg",
		"-i", vad.audioFile,
		"-af", "volumedetect",
		"-f", "null",
		"-",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0.0
	}

	// Parse the output for volume information
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "mean_volume:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "mean_volume:" && i+1 < len(parts) {
					if val, err := strconv.ParseFloat(parts[i+1], 64); err == nil {
						// Convert dB to linear scale (rough approximation)
						return (val + 60) / 60.0 // Normalize -60dB to 0dB range to 0-1
					}
				}
			}
		}
	}

	return 0.0
}
