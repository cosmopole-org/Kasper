package firectl

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"kasper/src/shell/utils/future"
	"os"
	"sync"
	"time"

	adapters "kasper/src/abstract/adapters/firectl"

	firecracker "github.com/firecracker-microvm/firecracker-go-sdk"
	models "github.com/firecracker-microvm/firecracker-go-sdk/client/models"
	cmap "github.com/orcaman/concurrent-map/v2"
	log "github.com/sirupsen/logrus"
)

type ChannelWriteCloser struct {
	C      chan string
	closed bool
}

// Write sends the string to the channel
func (w *ChannelWriteCloser) Write(p []byte) (int, error) {
	if w.closed {
		return 0, fmt.Errorf("write to closed ChannelWriteCloser")
	}
	w.C <- string(p)
	return len(p), nil
}

// Close closes the channel
func (w *ChannelWriteCloser) Close() error {
	if !w.closed {
		close(w.C)
		w.closed = true
	}
	return nil
}

type ChannelReader struct {
	C    chan []byte
	buf  *bytes.Buffer
	done bool
}

func (r *ChannelReader) Read(p []byte) (int, error) {
	for r.buf == nil || r.buf.Len() == 0 {
		chunk, ok := <-r.C
		if !ok {
			r.done = true
			break
		}
		r.buf = bytes.NewBuffer(chunk)
	}

	if r.done && (r.buf == nil || r.buf.Len() == 0) {
		return 0, io.EOF
	}

	return r.buf.Read(p)
}

type TerminalManager struct {
	writer       *ChannelWriteCloser
	reader       *ChannelReader
	mu           sync.Mutex
	outputBuffer []byte
	listeners    []chan string
	done         chan struct{}
	logger       *log.Entry
}

type FireCtl struct {
	VMs *cmap.ConcurrentMap[string, *adapters.VM]
}

func NewFireCtl() *FireCtl {
	m := cmap.New[*adapters.VM]()
	return &FireCtl{VMs: &m}
}

func init() {
	// Configure logrus
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)
}

// NewTerminalManager creates a new terminal manager
func NewTerminalManager(writer *ChannelWriteCloser, reader *ChannelReader, logger *log.Entry) *TerminalManager {
	return &TerminalManager{
		writer: writer,
		reader: reader,
		done:   make(chan struct{}),
		logger: logger.WithField("component", "terminal"),
	}
}

// Start begins reading from the terminal and notifying listeners
func (tm *TerminalManager) Start() {
	tm.logger.Info("Starting terminal output reader")
	go tm.readTerminalOutput()
}

// Stop terminates terminal reading
func (tm *TerminalManager) Stop() {
	close(tm.done)
	tm.writer.Close()
}

// readTerminalOutput continuously reads from the console
func (tm *TerminalManager) readTerminalOutput() {
	reader := bufio.NewReader(tm.reader)
	for {
		select {
		case <-tm.done:
			tm.logger.Debug("Terminal reader exiting")
			return
		default:
			buf := make([]byte, 1024)
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					tm.logger.Errorf("Terminal read error: %v", err)
				}
				return
			}
			if n > 0 {
				tm.processOutput(buf[:n])
			}
		}
	}
}

// processOutput handles new output and notifies listeners
func (tm *TerminalManager) processOutput(data []byte) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Add to buffer
	tm.outputBuffer = append(tm.outputBuffer, data...)

	// Convert to string and notify listeners
	output := string(data)
	for _, ch := range tm.listeners {
		select {
		case ch <- output:
		default:
			tm.logger.Warn("Output listener channel full, dropping data")
		}
	}
}

// GetOutput returns the current output buffer
func (tm *TerminalManager) GetOutput() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return string(tm.outputBuffer)
}

// ClearOutput clears the output buffer
func (tm *TerminalManager) ClearOutput() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.outputBuffer = []byte{}
	tm.logger.Info("Terminal output cleared")
}

// SendCommand sends a command to the VM
func (tm *TerminalManager) SendCommand(cmd string) {
	tm.reader.C <- []byte(cmd + "\n")
	tm.logger.WithField("command", cmd).Debug("Command sent to VM")
}

// RegisterListener adds a new output listener
func (tm *TerminalManager) RegisterListener() <-chan string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	ch := make(chan string, 100) // Larger buffer for high output volume
	tm.listeners = append(tm.listeners, ch)
	tm.logger.Debug("New output listener registered")
	return ch
}

// RemoveListener removes an output listener
func (tm *TerminalManager) RemoveListener(ch <-chan string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i, listener := range tm.listeners {
		if listener == ch {
			// Remove without preserving order
			tm.listeners[i] = tm.listeners[len(tm.listeners)-1]
			tm.listeners = tm.listeners[:len(tm.listeners)-1]
			close(listener)
			tm.logger.Debug("Output listener removed")
			break
		}
	}
}

func (f *FireCtl) StopVm(id string) {
	future.Async(func() {
		vm, found := f.VMs.Get(id)
		if !found {
			return
		}
		vm.SigCh <- 1
	}, false)
}

func (f *FireCtl) GetVm(id string) *adapters.VM {
	vm, found := f.VMs.Get(id)
	if found {
		return vm
	}
	return nil
}

func (f *FireCtl) RunVm(id string, terminal chan string) {
	future.Async(func() {
		socketPath := "/opt/firecracker/vms/fc." + id + ".sock"
		const (
			kernelPath = "/opt/firecracker/kernel/vmlinux"
			rootfsPath = "/opt/firecracker/rootfs/rootfs.ext4"
		)

		// Create logger
		logger := log.New()
		logger.SetFormatter(&log.JSONFormatter{})
		logger.SetLevel(log.InfoLevel)

		// Log to file and stdout
		logFile, err := os.Create("/opt/firecracker/vms/fc.log")
		if err != nil {
			logger.Fatalf("Failed to create log file: %v", err)
		}
		defer logFile.Close()

		multiWriter := io.MultiWriter(os.Stdout, logFile)
		logger.SetOutput(multiWriter)

		mainLog := logger.WithFields(log.Fields{
			"component": "main",
			"vm_id":     "firecracker-vm-" + id,
		})

		mainLog.Info("Starting Firecracker VM manager")
		ctx := context.Background()

		// Enhanced VM configuration
		cfg := firecracker.Config{
			SocketPath:      socketPath,
			KernelImagePath: kernelPath,
			KernelArgs:      "console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda ip=dhcp",
			Drives: []models.Drive{{
				DriveID:      firecracker.String("rootfs"),
				PathOnHost:   firecracker.String(rootfsPath),
				IsRootDevice: firecracker.Bool(true),
				IsReadOnly:   firecracker.Bool(false),
			}},
			MachineCfg: models.MachineConfiguration{
				VcpuCount:       firecracker.Int64(1),
				MemSizeMib:      firecracker.Int64(512),
				Smt:             firecracker.Bool(false),
				TrackDirtyPages: true,
			},
		}

		// Handle signals
		sigCh := make(chan int, 1)

		writerChannel := make(chan string)
		readerChannel := make(chan []byte)

		// Create terminal manager
		termManager := NewTerminalManager(&ChannelWriteCloser{C: writerChannel}, &ChannelReader{C: readerChannel}, mainLog)
		termManager.Start()
		defer termManager.Stop()

		log.Println("created terminal manager.")

		// Configure VM command
		cmd := firecracker.VMCommandBuilder{}.
			WithBin("/usr/local/bin/firecracker").
			WithSocketPath(socketPath).
			WithStdin(termManager.reader).
			WithStdout(termManager.writer).
			WithStderr(termManager.writer).
			Build(ctx)

		log.Println("created vm commander.")

		// Create logrus logger for Firecracker SDK
		fcLogger := log.New()
		fcLogger.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
		fcLogger.SetOutput(termManager.writer)
		fcLogger.SetLevel(log.WarnLevel) // Reduce SDK log noise

		log.Println("created vm logger.")

		// Create machine
		machine, err := firecracker.NewMachine(ctx, cfg,
			firecracker.WithProcessRunner(cmd),
			firecracker.WithLogger(log.NewEntry(fcLogger)),
		)
		if err != nil {
			mainLog.Fatalf("Failed to create machine: %v", err)
		}

		log.Println("created the machine.")

		f.VMs.Set(id, &adapters.VM{Machine: machine, Terminal: termManager, SigCh: sigCh})

		// Start VM in goroutine
		go func() {
			if err := machine.Start(ctx); err != nil {
				mainLog.Fatalf("Failed to start VM: %v", err)
			}
			mainLog.Info("VM started successfully")

			// Wait for VM to exit
			if err := machine.Wait(ctx); err != nil {
				mainLog.Warnf("VM wait returned error: %v", err)
			}
		}()

		go func() {
			for output := range writerChannel {
				mainLog.WithFields(log.Fields{
					"source": "vm_output",
					"length": len(output),
				}).Debug(output)
				terminal <- output
			}
		}()

		// Wait for VM to boot
		mainLog.Info("Waiting for VM to boot...")

		log.Println("step 1 of executing vm...")

		time.Sleep(5 * time.Second)

		log.Println("step 2 of executing vm...")

		// Example: Send initial commands
		termManager.SendCommand("echo 'Hello from VM terminal'")

		log.Println("step 3 of executing vm...")

		termManager.SendCommand("uname -a")

		log.Println("step 4 of executing vm...")

		mainLog.Info("VM is running. Enter commands in this terminal")
		mainLog.Info("Special commands: exit, clear, snapshot, status, loglevel [debug|info]")
		mainLog.Info("Press Ctrl+C to shutdown VM")

		log.Println("step 5 of executing vm...")

		// Wait for shutdown signal
		<-sigCh
		mainLog.Info("Shutting down VM")

		log.Println("step 6 of executing vm...")

		// Create snapshot before shutdown
		f.CreateSnapshot(ctx, machine, mainLog)

		// Stop the VM
		if err := machine.StopVMM(); err != nil {
			mainLog.Errorf("Error stopping VM: %v", err)
		} else {
			mainLog.Info("VM stopped successfully")
		}
		terminal <- ""
	}, false)
}

func (f *FireCtl) CreateSnapshot(ctx context.Context, machine *firecracker.Machine, logger *log.Entry) {
	snapshotPath := "/opt/firecracker/snapshots/snapshot.bin"

	logger.Info("Pausing VM for snapshot")
	if err := machine.PauseVM(ctx); err == nil {
		logger.Info("Creating snapshot")
		if err := machine.CreateSnapshot(ctx, snapshotPath+".mem", snapshotPath); err != nil {
			logger.Errorf("Snapshot failed: %v", err)
		} else {
			logger.Info("Snapshot created successfully")
		}

		logger.Info("Resuming VM")
		if err := machine.ResumeVM(ctx); err != nil {
			logger.Errorf("Failed to resume VM: %v", err)
		}
	} else {
		logger.Errorf("Failed to pause VM for snapshot: %v", err)
	}
}
