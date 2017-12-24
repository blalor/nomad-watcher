package main

import (
    "os"
    "fmt"
    "syscall"
    "encoding/json"
    
    flags "github.com/jessevdk/go-flags"
    log "github.com/Sirupsen/logrus"
    
    nomad "github.com/hashicorp/nomad/api"

    "github.com/blalor/nomad-watcher/watcher"
)

var version string = "undef"

type Options struct {
    Debug bool       `env:"DEBUG"      long:"debug"      description:"enable debug"`
    LogFile string   `env:"LOG_FILE"   long:"log-file"   description:"path to JSON log file"`
    EventFile string `env:"EVENT_FILE" long:"event-file" description:"path to JSON event file" required:"true"`
}

func main() {
    var opts Options
    
    _, err := flags.Parse(&opts)
    if err != nil {
        os.Exit(1)
    }
    
    if opts.Debug {
        log.SetLevel(log.DebugLevel)
    }
    
    if opts.LogFile != "" {
        logFp, err := os.OpenFile(opts.LogFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0600)
        checkError(fmt.Sprintf("error opening %s", opts.LogFile), err)
        
        defer logFp.Close()
        
        // ensure panic output goes to log file
        syscall.Dup2(int(logFp.Fd()), 1)
        syscall.Dup2(int(logFp.Fd()), 2)
        
        // log as JSON
        log.SetFormatter(&log.JSONFormatter{})
        
        // send output to file
        log.SetOutput(logFp)
    }
    
    log.Debug("hi there! (tickertape tickertape)")
    log.Infof("version: %s", version)
    
    evtsFp, err := os.OpenFile(opts.EventFile, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0600)
    checkError(fmt.Sprintf("error opening %s", opts.EventFile), err)

    defer evtsFp.Close()

    nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
    checkError("creating Nomad client", err)
    
    enc := json.NewEncoder(evtsFp)
    
    eventChan := make(chan interface{})
    
    allocEventChan, taskStateEventChan := watcher.WatchAllocations(nomadClient.Allocations())
    go func() {
        for ae := range allocEventChan {
            eventChan <- ae
        }
    }()
    
    go func() {
        for tse := range taskStateEventChan {
            eventChan <- tse
        }
    }()
    
    go func() {
        for ee := range watcher.WatchEvaluations(nomadClient.Evaluations()) {
            eventChan <- ee
        }
    }()
    
    go func() {
        for je := range watcher.WatchJobs(nomadClient.Jobs()) {
            eventChan <- je
        }
    }()
    
    go func() {
        for ne := range watcher.WatchNodes(nomadClient.Nodes()) {
            eventChan <- ne
        }
    }()
    
    for e := range eventChan {
        checkError("serializing event", enc.Encode(e))
    }
}
