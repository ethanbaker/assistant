package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethanbaker/assistant/pkg/sdk"
	"github.com/ethanbaker/assistant/pkg/utils"
	"github.com/sashabaranov/go-openai"
)

const (
	// Session management constants
	SessionTimeout      = 10 * time.Minute // Time after which to create a new session
	MessageGroupTimeout = 5 * time.Minute  // Time to group messages in the same session

	// Audio constants
	AudioSampleRate = 16000            // Sample rate for audio recording
	AudioFileName   = "temp_audio.wav" // Temporary audio file

	// Voice Activity Detection constants
	SilenceThreshold   = 0.01 // Amplitude threshold for silence detection
	MinSpeechDuration  = 1.0  // Minimum duration of speech in seconds
	MaxSilenceDuration = 5.0  // Maximum silence duration before stopping in seconds
	AudioBufferSize    = 1024 // Audio buffer size for streaming
)

// VoiceAssistant represents the main voice assistant application
type VoiceAssistant struct {
	config       *utils.Config
	apiClient    *sdk.Client
	openaiClient *openai.Client

	// Session management
	currentSessionID   string
	lastMessageTime    time.Time
	sessionCreatedTime time.Time
}

// NewVoiceAssistant creates a new voice assistant instance
func NewVoiceAssistant(cfg *utils.Config) (*VoiceAssistant, error) {
	// Validate required configuration
	backendURL := cfg.Get("BACKEND_BASE_URL")
	if backendURL == "" {
		return nil, fmt.Errorf("BACKEND_BASE_URL not set in config or environment")
	}

	backendAPIKey := cfg.Get("BACKEND_API_KEY")
	if backendAPIKey == "" {
		return nil, fmt.Errorf("BACKEND_API_KEY not set in config or environment")
	}

	openaiAPIKey := cfg.Get("OPENAI_API_KEY")
	if openaiAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not set in config or environment")
	}

	// Create API clients
	apiClient := sdk.NewClient(backendURL, backendAPIKey)
	openaiClient := openai.NewClient(openaiAPIKey)

	return &VoiceAssistant{
		config:       cfg,
		apiClient:    apiClient,
		openaiClient: openaiClient,
	}, nil
}

// Start begins the voice assistant main loop
func (va *VoiceAssistant) Start(ctx context.Context) error {
	log.Println("[LISTEN]: Voice assistant started. Say something to begin...")

	// Check if required tools are available
	if err := va.checkDependencies(); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	// Main voice interaction loop
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := va.processVoiceInteraction(ctx); err != nil {
					log.Printf("[LISTEN]: Error processing voice interaction: %v", err)
					time.Sleep(2 * time.Second) // Brief pause before trying again
				}
			}
		}
	}()

	return nil
}

// Stop gracefully stops the voice assistant
func (va *VoiceAssistant) Stop() error {
	log.Println("[LISTEN]: Stopping voice assistant...")

	// Remove any temporary audio files
	tempAudio := filepath.Join(os.TempDir(), AudioFileName)
	if _, err := os.Stat(tempAudio); err == nil {
		return os.Remove(tempAudio)
	}

	return nil
}

// checkDependencies verifies that required external tools are available
func (va *VoiceAssistant) checkDependencies() error {
	// Check if ffmpeg is available for audio conversion
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found - required for audio processing")
	}

	// Check if arecord is available for recording (Linux)
	if _, err := exec.LookPath("arecord"); err != nil {
		log.Println("[LISTEN]: Warning - arecord not found, will try alternative recording methods")
	}

	// Check if SoX is available for enhanced voice activity detection
	if _, err := exec.LookPath("sox"); err != nil {
		log.Println("[LISTEN]: Warning - SoX not found, will use basic voice activity detection")
	}

	// Check if espeak/festival is available for text-to-speech
	if _, err := exec.LookPath("espeak-ng"); err != nil {
		log.Println("[LISTEN]: Warning - espeak-ng not found for TTS")
	}

	return nil
}

// processVoiceInteraction handles one complete voice interaction cycle
func (va *VoiceAssistant) processVoiceInteraction(ctx context.Context) error {
	log.Println("[LISTEN]: Listening for speech... (Speak now)")

	// Record audio from user with voice activity detection
	audioFile, err := va.recordAudio(ctx)
	if err != nil {
		return fmt.Errorf("failed to record audio: %w", err)
	}
	defer os.Remove(audioFile) // Clean up audio file

	// Check if we actually got some audio content
	if info, err := os.Stat(audioFile); err != nil || info.Size() < 1000 { // Less than 1KB suggests no real audio
		log.Println("[LISTEN]: No significant audio detected, listening again...")
		time.Sleep(1 * time.Second) // Brief pause before next attempt
		return nil
	}

	log.Println("[LISTEN]: Audio captured, transcribing...")

	// Transcribe audio to text using Whisper
	transcription, err := va.transcribeAudio(ctx, audioFile)
	if err != nil {
		return fmt.Errorf("failed to transcribe audio: %w", err)
	}

	// Clean up transcription (remove extra whitespace, etc.)
	transcription = strings.TrimSpace(transcription)

	if transcription == "" || len(transcription) < 3 {
		log.Println("[LISTEN]: No clear speech detected in transcription, listening again...")
		time.Sleep(1 * time.Second)
		return nil
	}

	log.Printf("[LISTEN]: User said: %s", transcription)

	// Get or create session
	sessionID, err := va.getOrCreateSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to get/create session: %w", err)
	}

	// Send message to assistant API
	response, err := va.sendMessage(ctx, sessionID, transcription)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	log.Printf("[LISTEN]: Assistant: %s", response)

	// Convert response to speech and play it
	if err := va.speakText(ctx, response); err != nil {
		log.Printf("[LISTEN]: Warning - failed to speak response: %v", err)
		// Don't return error here, as the text response was still provided
	}

	// Update last message time
	va.lastMessageTime = time.Now()

	// Brief pause before listening again
	time.Sleep(2 * time.Second)

	return nil
}

// recordAudio streams audio from the microphone with voice activity detection
func (va *VoiceAssistant) recordAudio(ctx context.Context) (string, error) {
	audioFile := filepath.Join(os.TempDir(), AudioFileName)

	// Create a context with timeout as a safety measure (max 60 seconds)
	recordCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Use arecord for streaming audio with voice activity detection
	if _, err := exec.LookPath("arecord"); err == nil {
		return va.recordWithArecord(recordCtx, audioFile)
	}

	// Fallback to ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err == nil {
		return va.recordWithFFmpeg(recordCtx, audioFile)
	}

	return "", fmt.Errorf("no audio recording tool available (tried arecord, ffmpeg)")
}

// recordWithArecord records audio using arecord with streaming and VAD
func (va *VoiceAssistant) recordWithArecord(ctx context.Context, audioFile string) (string, error) {
	// Start arecord in streaming mode (no duration limit)
	cmd := exec.CommandContext(ctx, "arecord",
		"-D", "default", // Default audio device
		"-f", "S16_LE", // 16-bit little-endian format
		"-c", "1", // Mono
		"-r", fmt.Sprintf("%d", AudioSampleRate), // Sample rate
		audioFile,
	)

	// Start the recording process
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start arecord: %w", err)
	}

	// Use the enhanced VAD system
	vad := NewVoiceActivityDetector(audioFile)

	// Monitor for voice activity in a separate goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		vad.MonitorWithSoX(ctx, cmd)
	}()

	// Wait for either VAD completion or context cancellation
	select {
	case <-done:
		// VAD detected end of speech
	case <-ctx.Done():
		// Context was cancelled
	}

	// Stop the recording process
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	cmd.Wait()

	return audioFile, nil
}

// recordWithFFmpeg records audio using ffmpeg with streaming and VAD
func (va *VoiceAssistant) recordWithFFmpeg(ctx context.Context, audioFile string) (string, error) {
	// Start ffmpeg in streaming mode
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "pulse", // Use PulseAudio
		"-i", "default", // Default input device
		"-ar", fmt.Sprintf("%d", AudioSampleRate), // Sample rate
		"-ac", "1", // Mono
		"-y", // Overwrite output file
		audioFile,
	)

	// Start the recording process
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	// Use the enhanced VAD system
	vad := NewVoiceActivityDetector(audioFile)

	// Monitor for voice activity in a separate goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		vad.MonitorWithSoX(ctx, cmd)
	}()

	// Wait for either VAD completion or context cancellation
	select {
	case <-done:
		// VAD detected end of speech
	case <-ctx.Done():
		// Context was cancelled
	}

	// Stop the recording process
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	cmd.Wait()

	return audioFile, nil
}

// transcribeAudio uses OpenAI Whisper to convert audio to text
func (va *VoiceAssistant) transcribeAudio(ctx context.Context, audioFile string) (string, error) {
	// Open audio file
	file, err := os.Open(audioFile)
	if err != nil {
		return "", fmt.Errorf("failed to open audio file: %w", err)
	}
	defer file.Close()

	// Create transcription request
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: audioFile,
		Language: "en",
	}

	// Call OpenAI API for transcription
	resp, err := va.openaiClient.CreateTranscription(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create transcription: %w", err)
	}

	return resp.Text, nil
}

// getOrCreateSession manages session lifecycle based on timing
func (va *VoiceAssistant) getOrCreateSession(ctx context.Context) (string, error) {
	now := time.Now()

	// Check if we need a new session
	needNewSession := va.currentSessionID == "" ||
		now.Sub(va.sessionCreatedTime) > SessionTimeout ||
		now.Sub(va.lastMessageTime) > MessageGroupTimeout

	if needNewSession {
		// Create a new session
		req := &sdk.CreateSessionRequest{
			UserID: "voice-user", // Can be made configurable
		}

		session, err := va.apiClient.CreateSession(ctx, req)
		if err != nil {
			return "", fmt.Errorf("failed to create session: %w", err)
		}

		va.currentSessionID = session.ID
		va.sessionCreatedTime = now
		log.Printf("[LISTEN]: Created new session: %s", session.ID)
	}

	return va.currentSessionID, nil
}

// sendMessage sends a message to the assistant API and returns the response
func (va *VoiceAssistant) sendMessage(ctx context.Context, sessionID, message string) (string, error) {
	req := &sdk.PostMessageRequest{
		Content: message,
	}

	resp, err := va.apiClient.SendMessage(ctx, sessionID, req)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	return resp.FinalOutput, nil
}

// speakText converts text to speech and plays it
func (va *VoiceAssistant) speakText(ctx context.Context, text string) error {
	// Try espeak first
	if _, err := exec.LookPath("espeak"); err == nil {
		cmd := exec.CommandContext(ctx, "espeak", "-s", "150", "-v", "en", text)
		return cmd.Run()
	}

	// Try festival as fallback
	if _, err := exec.LookPath("festival"); err == nil {
		cmd := exec.CommandContext(ctx, "festival", "--tts")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}

	// If no TTS available, just print the text
	fmt.Printf("[VOICE TTS]: %s\n", text)
	return nil
}
