package main

import (
    "os"
    "fmt"
    "strings"
    "reflect"

    flags "github.com/jessevdk/go-flags"

    nomad "github.com/hashicorp/nomad/api"

    "github.com/mgutz/ansi"

    "github.com/blalor/nomad-watcher/watcher"
)

var version string = "undef"

type Options struct {
    NoWatchAllocs     bool `long:"no-watch-allocs"`
    NoWatchTaskStates bool `long:"no-watch-task-states"`
    NoWatchEvals      bool `long:"no-watch-evals"`
    NoWatchJobs       bool `long:"no-watch-jobs"`
    NoWatchNodes      bool `long:"no-watch-nodes"`
}

var (
    c_node  = ansi.ColorFunc("yellow")  // node: yellow
    c_eval  = ansi.ColorFunc("cyan")    // eval: blue
    c_task  = ansi.ColorFunc("red")     // task: red
    c_alloc = ansi.ColorFunc("green")   // alloc: green
    c_job   = ansi.ColorFunc("magenta") // job: magenta
)

func trimId(id string) string {
    return strings.Split(id, "-")[0]
}

func main() {
    var opts Options

    _, err := flags.Parse(&opts)
    if err != nil {
        os.Exit(1)
    }

    nomadClient, err := nomad.NewClient(nomad.DefaultConfig())
    checkError("creating Nomad client", err)

    eventChan := make(chan interface{})

    if ! opts.NoWatchAllocs || ! opts.NoWatchTaskStates {
        allocEventChan, taskStateEventChan := watcher.WatchAllocations(nomadClient.Allocations())

        go func() {
            for ae := range allocEventChan {
                if ! opts.NoWatchAllocs {
                    eventChan <- ae
                }
            }
        }()

        go func() {
            for tse := range taskStateEventChan {
                if ! opts.NoWatchTaskStates {
                    eventChan <- tse
                }
            }
        }()
    }

    if ! opts.NoWatchEvals {
        go func() {
            for ee := range watcher.WatchEvaluations(nomadClient.Evaluations()) {
                eventChan <- ee
            }
        }()
    }

    if ! opts.NoWatchJobs {
        go func() {
            for je := range watcher.WatchJobs(nomadClient.Jobs()) {
                eventChan <- je
            }
        }()
    }

    if ! opts.NoWatchNodes {
        go func() {
            for ne := range watcher.WatchNodes(nomadClient.Nodes()) {
                eventChan <- ne
            }
        }()
    }

    for e := range eventChan {
        switch typ := e.(type) {
            case watcher.AllocEvent:
                a := e.(watcher.AllocEvent).AllocationListStub

                // <alloc> A[59f464da] E[1e914a0b] N[9d451b2b] example     stop         'alloc not needed due to job update' running      ''
                fmt.Printf(
                    "%s A[%s] E[%s] N[%s] %-20s %-12s '%s' %-12s '%s'\n",

                    c_alloc("<alloc>"),
                    c_alloc(trimId(a.ID)),
                    c_eval(trimId(a.EvalID)),
                    c_node(trimId(a.NodeID)),
                    c_job(a.JobID),
                    a.DesiredStatus,
                    a.DesiredDescription,
                    a.ClientStatus,
                    a.ClientDescription,
                )

            case watcher.TaskStateEvent:
                t := e.(watcher.TaskStateEvent)
                te := t.TaskEvent

                // only output key/value pairs of TaskEvent that are non-empty strings
                var err_strs []string
                rVal := reflect.ValueOf(*te)
                for i := 0; i < rVal.NumField(); i++ {
                    field := rVal.Field(i)

                    if field.Type() == reflect.TypeOf("") {
                        structName := rVal.Type().Field(i).Name
                        structVal := field.Interface().(string)

                        if ! (structName == "TaskSignal" || structName == "Type") && structVal != "" {
                            err_strs = append(err_strs, structName + ": " + structVal)
                        }
                    }
                }

                failedStr := "N"
                if t.Failed {
                    failedStr = "Y"
                }

                // <task > A[7a5be77d] example.cache[0] pending      failed? N Driver DriverMessage: Downloading image redis:3.2
                fmt.Printf(
                    "%s A[%s] %-20s %-12s failed? %s '%s' %s\n",

                    c_task("<task >"),
                    c_alloc(trimId(t.AllocID)),
                    c_alloc(t.AllocName),
                    t.State,
                    failedStr,
                    te.Type,
                    strings.Join(err_strs, ", "),
                )

            case watcher.EvalEvent:
                e := e.(watcher.EvalEvent).Evaluation

                // <eval > E[a0dfc1cf] deployment-watcher example     N[        ] complete     next: E[        ] prev: E[        ] block: E[        ]
                fmt.Printf(
                    "%s E[%s] %-14s %-20s N[%-8s] %-12s next: E[%-8s] prev: E[%-8s] block: E[%-8s]\n",

                    c_eval("<eval >"),
                    c_eval(trimId(e.ID)),
                    e.TriggeredBy,
                    c_job(e.JobID),
                    c_node(trimId(e.NodeID)),
                    e.Status,
                    c_eval(trimId(e.NextEval)),
                    c_eval(trimId(e.PreviousEval)),
                    c_eval(trimId(e.BlockedEval)),
                )


            case watcher.JobEvent:
                j := e.(watcher.JobEvent).JobListStub

                // <job  > service 050 example     pending
                fmt.Printf(
                    "%s %s %03d %-20s %s\n",

                    c_job("<job  >"),
                    j.Type,
                    j.Priority,
                    c_job(j.ID),
                    j.Status,
                )

            case watcher.NodeEvent:
                n := e.(watcher.NodeEvent).NodeListStub

                // <node > N[9d451b2b] Scooter.fios-router.home initializing false
                fmt.Printf(
                    "%s N[%-8s] %s %-12s %-5t\n",

                    c_node("<node >"),
                    c_node(trimId(n.ID)),
                    c_node(n.Name),
                    n.Status,
                    n.Drain,
                )
            default:
                fmt.Printf("unexpected type %T\n", typ)
        }
    }
}
