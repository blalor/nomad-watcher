package main

import (
    "os"
    "fmt"
    "strings"
    "reflect"
    "math"

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
    NoWatchDeploys    bool `long:"no-watch-deploys"`
}

var (
    c_node   = ansi.ColorFunc("yellow")
    c_eval   = ansi.ColorFunc("blue")
    c_task   = ansi.ColorFunc("red")
    c_alloc  = ansi.ColorFunc("green")
    c_job    = ansi.ColorFunc("magenta")
    c_deploy = ansi.ColorFunc("cyan")
    c_under  = ansi.ColorFunc("white+u")
    c_fail   = ansi.ColorFunc("white+b:red+h")
)

var (
    TMPL_ALLOC_ID  = "A[" + c_alloc("%-8s") + "]"
    TMPL_EVAL_ID   = "E[" + c_eval("%-8s") + "]"
    TMPL_NODE_ID   = "N[" + c_node("%-8s") + "]"
    TMPL_DEPLOY_ID = "D[" + c_node("%-8s") + "]"

    TMPL_JOB_ID = func(jobIdLen float64) string {
        return c_job(fmt.Sprintf("%%-%ds", int(jobIdLen)))
    }

    TMPL_ALLOC_NAME = func(allocNameLen float64) string {
        return c_alloc(fmt.Sprintf("%%-%ds", int(allocNameLen)))
    }

    TMPL_NODE_NAME = func(nodeNameLen float64) string {
        return c_node(fmt.Sprintf("%%-%ds", int(nodeNameLen)))
    }
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

    if ! opts.NoWatchDeploys {
        go func() {
            for de := range watcher.WatchDeployments(nomadClient.Deployments()) {
                eventChan <- de
            }
        }()
    }

    jobIdLen := float64(0)
    allocNameLen := float64(0)
    nodeNameLen := float64(0)
    for e := range eventChan {
        switch typ := e.(type) {
            case watcher.AllocEvent:
                a := e.(watcher.AllocEvent).Allocation
                jobIdLen = math.Max(jobIdLen, float64(len(a.JobID)))

                // <alloc> A[59f464da] example stop 'alloc not needed due to job update' running '' E[1e914a0b] N[9d451b2b]
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_alloc("<alloc >"),
                            TMPL_ALLOC_ID,
                            TMPL_JOB_ID(jobIdLen),
                            c_job("v%03d"),
                            TMPL_EVAL_ID,
                            TMPL_NODE_ID,
                            c_under("%s"),
                            "%s",
                            c_under("%s"),
                            "%s\n",
                        },
                        " ",
                    ),

                    trimId(a.ID),
                    a.JobID,
                    *a.Job.Version,
                    trimId(a.EvalID),
                    trimId(a.NodeID),
                    a.DesiredStatus,
                    a.DesiredDescription,
                    a.ClientStatus,
                    a.ClientDescription,
                )


            case watcher.TaskStateEvent:
                t := e.(watcher.TaskStateEvent).TaskState
                allocNameLen = math.Max(allocNameLen, float64(len(t.AllocName)))

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
                            err_strs = append(err_strs, c_under(structName) + "=" + structVal)
                        }
                    }
                }

                failedStr := ""
                if t.Failed {
                    failedStr = c_fail("FAIL")
                }

                // <task > A[7a5be77d] example.cache[0] pending      FAIL Driver DriverMessage: Downloading image redis:3.2
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_task("<task  >"),
                            TMPL_ALLOC_ID,
                            TMPL_ALLOC_NAME(allocNameLen),
                            "%-12s %s %s %s\n",
                        },
                        " ",
                    ),

                    trimId(t.AllocID),
                    t.AllocName,
                    t.State,
                    failedStr,
                    te.Type,
                    strings.Join(err_strs, ", "),
                )

            case watcher.EvalEvent:
                e := e.(watcher.EvalEvent).Evaluation
                jobIdLen = math.Max(jobIdLen, float64(len(e.JobID)))

                // <eval > E[a0dfc1cf] example deployment-watcher D[        ] N[        ] complete     next: E[        ] prev: E[        ] block: E[        ]
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_eval("<eval  >"),
                            TMPL_EVAL_ID,
                            TMPL_JOB_ID(jobIdLen),
                            "%-14s",
                            "%-12s",
                            TMPL_DEPLOY_ID,
                            TMPL_NODE_ID,
                            "next: " + TMPL_EVAL_ID,
                            "prev: " + TMPL_EVAL_ID,
                            "block: " + TMPL_EVAL_ID + "\n",
                        },
                        " ",
                    ),

                    trimId(e.ID),
                    e.JobID,
                    e.TriggeredBy,
                    e.Status,
                    trimId(e.DeploymentID),
                    trimId(e.NodeID),
                    trimId(e.NextEval),
                    trimId(e.PreviousEval),
                    trimId(e.BlockedEval),
                )

            case watcher.JobEvent:
                j := e.(watcher.JobEvent).Job
                jobIdLen = math.Max(jobIdLen, float64(len(*j.ID)))

                // <job  > example service 050      pending
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_job("<job   >"),
                            TMPL_JOB_ID(jobIdLen),
                            c_job("v%03d"),
                            "%s %03d %s\n",
                        },
                        " ",
                    ),

                    *j.ID,
                    *j.Version,
                    *j.Type,
                    *j.Priority,
                    *j.Status,
                )

            case watcher.NodeEvent:
                n := e.(watcher.NodeEvent).NodeListStub
                nodeNameLen = math.Max(nodeNameLen, float64(len(n.Name)))

                // <node > N[9d451b2b] Scooter.fios-router.home initializing false
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_node("<node  >"),
                            TMPL_NODE_ID,
                            TMPL_NODE_NAME(nodeNameLen),
                            "%s %-5t\n",
                        },
                        " ",
                    ),

                    trimId(n.ID),
                    n.Name,
                    n.Status,
                    n.Drain,
                )


            case watcher.DeployEvent:
                d := e.(watcher.DeployEvent).Deployment
                jobIdLen = math.Max(jobIdLen, float64(len(d.JobID)))

                // <deploy> D[b17cb6a9] example     v253 running Deployment is running
                fmt.Printf(
                    strings.Join(
                        []string{
                            c_node("<deploy>"),
                            TMPL_DEPLOY_ID,
                            TMPL_JOB_ID(jobIdLen),
                            c_job("v%03d"),
                            "%s %s\n",
                        },
                        " ",
                    ),

                    trimId(d.ID),
                    d.JobID,
                    d.JobVersion,
                    d.Status,
                    d.StatusDescription,
                )

            default:
                fmt.Printf("unexpected type %T: %#v\n", typ, e)
        }
    }
}
