package main

import (
  "fmt"
  "flag"
  "net"
  "os"
  "os/signal"
  "os/exec"
  "syscall"
  "time"
  "runtime"
  "strings"
  "strconv"
  "path/filepath"
)

var (
  optFD = flag.Int("f", -1, "Set FD passed by parent process." )
  optName = flag.String("n", "", "Set name passed by parent process.")
  gListener net.Listener
  runningPID int
  passingFD = -1
)

func main() {
  runningPID = os.Getpid()
  flag.Parse()
  var err error
  gListener, err = newListener("tcp", "127.0.0.1:22222")
  if err != nil {
    LogErr(err.Error())
  }

  Log("Process[%d] started, waiting for signal!", runningPID)

  cSignal := make(chan os.Signal, 1)
  signal.Notify(cSignal, syscall.SIGHUP)
  // Block until a signal is received.
  for s := range cSignal {
    Log("Got signal: %v", s)
    switch (s) {
    case syscall.SIGHUP:
        Log("Action: Exit")
        newOSFD := dupNetFD()
        stopListening()
        startNewInstance(newOSFD)
        os.Exit(0)
      }
    }
}

func newListener(netType, laddr string) (net.Listener, error) {
  var l net.Listener
  var err error
  if *optFD > 0 && *optName != "" {
    Log("Using passed FD: %d, name: %s", *optFD, *optName)
    f := os.NewFile(uintptr(*optFD), *optName)
    l, err = net.FileListener(f)
    if err != nil {
      LogErr(err.Error())
      LogErr("Using net.Listen() instead")
      l, err = net.Listen(netType, laddr)
    }
  } else {
    l, err = net.Listen(netType, laddr)
  }

  return l, err
}

func dupNetFD() (newOSFD *os.File) {
  l := gListener.(*net.TCPListener)
  newOSFD, err := l.File() // net.TCPListener.File() call dup() to return a new FD
  
  if err == nil {
    newFD := newOSFD.Fd()
    name := newOSFD.Name()
    Log("New fd: " + strconv.Itoa(int(newFD)) + " Name: " + name)
  } else {
    LogErr(err.Error())
  }
  
  return newOSFD
}

func startNewInstance(newOSFD *os.File) {
  exec_path, _ := exec.LookPath(os.Args[0])
  path, _ := filepath.Abs(exec_path)
  args := make([]string, 0)
  args = append(args, fmt.Sprintf("-f=%d", newOSFD.Fd()))
  args = append(args, fmt.Sprintf("-n=%s", newOSFD.Name()))

  cmd := exec.Command(path, args...)
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  cmd.ExtraFiles = []*os.File{newOSFD}
  
  err := cmd.Start()
  if err != nil {
    LogErr(err.Error())
  }
}

func stopListening() {
  gListener.Close()
}

func Log(format string, args ...interface{}) {
  now := time.Now()
  year, month, day := now.Date()
  hour, minute, second := now.Clock()
  time_str := fmt.Sprintf("[GOZD][%d-%d-%d %d:%d:%d]", year, month, day, hour, minute, second)
  
  var nameStr string
  pidStr := fmt.Sprintf("[%d]", runningPID)
  pc, _, _, _ := runtime.Caller(1)
  name := runtime.FuncForPC(pc).Name()
  names := strings.Split(name, ".")
  if len(names) > 0 {
    nameStr = names[len(names)-1]
  }
  callerStr := "[" + nameStr + "] "
  fmt.Printf(time_str + pidStr + callerStr + format + "\n", args...)
}

func LogErr(format string, args ...interface{}) {
  now := time.Now()
  year, month, day := now.Date()
  hour, minute, second := now.Clock()
  time_str := fmt.Sprintf("[GOZDERR][%d-%d-%d %d:%d:%d]", year, month, day, hour, minute, second)
  
  var fileStr, nameStr string
  pidStr := fmt.Sprintf("[%d]", runningPID)
  pc, _, _, _ := runtime.Caller(1)
  file, line := runtime.FuncForPC(pc).FileLine(pc)
  files := strings.Split(file, "/")
  if len(files) > 0 {
    fileStr = files[len(files)-1]
  }
  name := runtime.FuncForPC(pc).Name()
  names := strings.Split(name, ".")
  if len(names) > 0 {
    nameStr = names[len(names)-1]
  }
  callerStr := "[" + nameStr + " " + fileStr + ":" + strconv.Itoa(line) + "] "
  fmt.Printf(time_str + pidStr + callerStr + format + "\n", args...)
}
