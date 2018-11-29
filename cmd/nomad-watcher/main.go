package main

import (
    "os"
    "fmt"
    "time"
    "syscall"
    "encoding/json"

    flags "github.com/jessevdk/go-flags"
    log "github.com/sirupsen/logrus"

    nomad "github.com/hashicorp/nomad/api"

    "github.com/blalor/nomad-watcher/watcher"
)

var version string = "undef"

type Options struct {
    Debug bool       `env:"DEBUG"      long:"debug"      description:"enable debug"`
    LogFile string   `env:"LOG_FILE"   long:"log-file"   description:"path to JSON log file"`
    EventFile string `env:"EVENT_FILE" long:"event-file" description:"path to JSON event file" required:"true"`
}

type event struct {
    Timestamp time.Time   `json:"@timestamp"`
    WaitIndex uint64      `json:"wait_index"`

    // consistent properties to make filtering events easier without needing
    // logic about the event payload
    Type         string `json:"type"`
    AllocationID string `json:"AllocationID,omitempty"`
    EvaluationID string `json:"EvaluationID,omitempty"`
    JobID        string `json:"JobID,omitempty"`
    NodeID       string `json:"NodeID,omitempty"`
    DeploymentID string `json:"DeploymentID,omitempty"`

    Event interface{} `json:"event"`
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
            eventChan <- &event{
                Timestamp: ae.Timestamp,
                WaitIndex: ae.WaitIndex,

                Type: "alloc",
                AllocationID: ae.Allocation.ID,
                EvaluationID: ae.Allocation.EvalID,
                JobID: ae.Allocation.JobID,
                NodeID: ae.Allocation.NodeID,
                DeploymentID: ae.Allocation.DeploymentID,

                Event: ae.Allocation,
            }
        }
    }()

    go func() {
        for tse := range taskStateEventChan {
            eventChan <- &event{
                Timestamp: tse.Timestamp,
                WaitIndex: tse.WaitIndex,

                Type: "task_state",
                AllocationID: tse.TaskState.AllocID,
                EvaluationID: tse.TaskState.EvalID,
                JobID: tse.TaskState.JobID,
                NodeID: tse.TaskState.NodeID,
                DeploymentID: tse.TaskState.DeploymentID,

                Event: tse.TaskState,
            }
        }
    }()

    go func() {
        for ee := range watcher.WatchEvaluations(nomadClient.Evaluations()) {
            eventChan <- &event{
                Timestamp: ee.Timestamp,
                WaitIndex: ee.WaitIndex,

                Type: "eval",
                AllocationID: "", // evals beget allocs
                EvaluationID: ee.Evaluation.ID,
                JobID: ee.Evaluation.JobID,
                NodeID: ee.Evaluation.NodeID,
                DeploymentID: ee.Evaluation.DeploymentID,

                Event: ee.Evaluation,
            }
        }
    }()

    go func() {
        for je := range watcher.WatchJobs(nomadClient.Jobs()) {
            eventChan <- &event{
                Timestamp: je.Timestamp,
                WaitIndex: je.WaitIndex,

                Type: "job",
                AllocationID: "",
                EvaluationID: "",
                JobID: *je.Job.ID,
                NodeID: "",
                DeploymentID: "",

                Event: je.Job,
            }
        }
    }()

    go func() {
        for ne := range watcher.WatchNodes(nomadClient.Nodes()) {
            eventChan <- &event{
                Timestamp: ne.Timestamp,
                WaitIndex: ne.WaitIndex,

                Type: "node",
                AllocationID: "",
                EvaluationID: "",
                JobID: "",
                NodeID: ne.Node.ID,
                DeploymentID: "",

                Event: ne.Node,
            }
        }
    }()

    go func() {
        for de := range watcher.WatchDeployments(nomadClient.Deployments()) {
            eventChan <- &event{
                Timestamp: de.Timestamp,
                WaitIndex: de.WaitIndex,

                Type: "deploy",
                AllocationID: "",
                EvaluationID: "",
                JobID: de.Deployment.JobID,
                NodeID: "",
                DeploymentID: de.Deployment.ID,

                Event: de.Deployment,
            }
        }
    }()

    for e := range eventChan {
        checkError("serializing event", enc.Encode(e))
    }
}
