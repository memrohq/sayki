// Package audio captures microphone input as 16 kHz mono float32 via miniaudio
// (malgo). miniaudio converts the device's native rate/channels for us, and
// follows system default-device changes when no explicit device is selected.
package audio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"

	"github.com/gen2brain/malgo"
)

const SampleRate = 16000

type Device struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"isDefault"`
}

type System struct {
	mu  sync.Mutex
	ctx *malgo.AllocatedContext
}

func NewSystem() (*System, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("audio init: %w", err)
	}
	return &System{ctx: ctx}, nil
}

func (s *System) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctx.Uninit()
	s.ctx.Free()
}

func (s *System) devices() ([]malgo.DeviceInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx.Devices(malgo.Capture)
}

func (s *System) Microphones() ([]Device, error) {
	infos, err := s.devices()
	if err != nil {
		return nil, err
	}
	out := make([]Device, 0, len(infos))
	for _, d := range infos {
		out = append(out, Device{ID: d.ID.String(), Name: d.Name(), IsDefault: d.IsDefault != 0})
	}
	return out, nil
}

// Capture is a running microphone stream.
type Capture struct {
	dev      *malgo.Device
	stopping atomic.Bool
	once     sync.Once
	Samples  chan []float32 // closed when the capture ends
	Errors   chan error     // device lost etc.; buffered, never closed
	Level    atomic.Uint32  // last chunk RMS as float32 bits, for the waveform
	Dropped  atomic.Int64
}

// Start opens deviceID ("" = system default, following changes) and streams samples.
func (s *System) Start(deviceID string) (*Capture, error) {
	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatF32
	cfg.Capture.Channels = 1
	cfg.SampleRate = SampleRate
	cfg.PeriodSizeInMilliseconds = 20

	if deviceID != "" {
		infos, err := s.devices()
		if err != nil {
			return nil, err
		}
		found := false
		for i := range infos {
			if infos[i].ID.String() == deviceID {
				cfg.Capture.DeviceID = infos[i].ID.Pointer()
				found = true
				break
			}
		}
		if !found {
			slog.Warn("selected microphone not found, using system default", "id", deviceID)
		}
	}

	c := &Capture{Samples: make(chan []float32, 256), Errors: make(chan error, 1)}
	cb := malgo.DeviceCallbacks{
		Data: func(_, in []byte, frames uint32) {
			buf := make([]float32, frames)
			var sum float64
			for i := range buf {
				v := math.Float32frombits(binary.LittleEndian.Uint32(in[4*i:]))
				buf[i] = v
				sum += float64(v * v)
			}
			c.Level.Store(math.Float32bits(float32(math.Sqrt(sum / float64(max(frames, 1))))))
			select { // never block the realtime audio thread
			case c.Samples <- buf:
			default:
				c.Dropped.Add(1)
			}
		},
		Stop: func() {
			if !c.stopping.Load() {
				select {
				case c.Errors <- errors.New("microphone stopped unexpectedly (disconnected?)"):
				default:
				}
			}
		},
	}
	s.mu.Lock()
	dev, err := malgo.InitDevice(s.ctx.Context, cfg, cb)
	s.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("open microphone: %w", err)
	}
	c.dev = dev
	if err := dev.Start(); err != nil {
		dev.Uninit()
		return nil, fmt.Errorf("start microphone: %w", err)
	}
	return c, nil
}

// Stop halts the device and closes Samples. Safe to call more than once.
func (c *Capture) Stop() {
	c.once.Do(func() {
		c.stopping.Store(true)
		c.dev.Uninit() // stops and waits for the audio thread, so no more sends
		close(c.Samples)
	})
}

func (c *Capture) RMS() float32 { return math.Float32frombits(c.Level.Load()) }
