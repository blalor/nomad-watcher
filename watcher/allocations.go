package watcher

import (
    "time"
    "github.com/sirupsen/logrus"

    "github.com/hashicorp/nomad/api"
)

type AllocEvent struct {
    Timestamp  time.Time       `json:"@timestamp"`
    WaitIndex  uint64          `json:"wait_index"`
    Allocation *api.Allocation `json:"alloc"`
}

type TaskState struct {
    // copied from the Allocation
    JobID              string // "elasticsearch-curator/periodic-1488845700"
    AllocID            string // "29cdfa9e-820a-bab8-4eda-45b000397719"
    AllocName          string // "elasticsearch-curator/periodic-1488845700.curator[0]"
    TaskGroup          string // "curator"
    EvalID             string // "8eb39148-7a00-a164-920f-d59143f72b74"
    NodeID             string // "f8fa01e0-351b-7652-d80e-10547fa3bfe8"
    DeploymentID       string // "2df116a0-4bc2-823d-4db7-a9959f42d6a0"

    // Task comes from the keys of TaskStates
    Task string

    // State and Failed come from the TaskState
    State  string
    Failed bool

    // the actual Task Event (phew!)
    TaskEvent *api.TaskEvent
}

// this is a derived event from an allocation. Time comes from TaskEvent.Time.
type TaskStateEvent struct {
    Timestamp  time.Time  `json:"@timestamp"`
    WaitIndex  uint64     `json:"wait_index"`
    TaskState  TaskState  `json:"task_state"`
}

func WatchAllocations(allocClient *api.Allocations) (<- chan AllocEvent, <- chan TaskStateEvent) {
    log := logrus.WithFields(logrus.Fields{
        "package": "watcher",
        "fn": "WatchAllocations",
    })

    allocEventChan := make(chan AllocEvent)
    taskStateEventChan := make(chan TaskStateEvent)

    keepWatching := true

    go func() {
        // keeping track of the most recently-seen task state event (TaskEvent)
        // timestamp for each allocation.  also allows for tracking allocations
        // that are removed.
        // @todo possibly need to track per-task times?
        allocTaskEventTimes := make(map[string]int64)

        queryOpts := &api.QueryOptions{
            WaitTime: 1 * time.Minute,
            AllowStale: true,
        }

        for keepWatching {
            log.Debugf("retrieving from index %d", queryOpts.WaitIndex)
            allocStubs, queryMeta, err := allocClient.List(queryOpts)

            if err != nil {
                log.Errorf("unable to list allocations: %v", err)
                continue
            }

            if queryOpts.WaitIndex > 0 {
                // only emit events after the first run; we're looking for
                // changes

                // map of allocation ids in the current result
                extantAllocs := make(map[string]bool)

                for _, allocStub := range allocStubs {
                    allocId := allocStub.ID
                    extantAllocs[allocId] = true

                    // DeploymentID is only available in the alloc
                    var deployId string

                    ts := time.Unix(0, allocStub.ModifyTime)
                    if (allocStub.CreateIndex == allocStub.ModifyIndex) {
                        // allocation just created; use CreateTime field
                        ts = time.Unix(0, allocStub.CreateTime)
                    }

                    // true if allocation was created or updated
                    allocationUpdated := (queryOpts.WaitIndex < allocStub.CreateIndex) || (queryOpts.WaitIndex < allocStub.ModifyIndex)

                    if allocationUpdated {
                        // retrieve alloc details so we can get job details
                        alloc, _, err := allocClient.Info(allocStub.ID, &api.QueryOptions{AllowStale: true})
                        if err != nil {
                            log.Errorf("unable to retrieve alloc for %s: %v", allocStub.ID, err)
                            continue
                        }

                        deployId = alloc.DeploymentID

                        // the alloc's ModifyIndex is often different. copy the
                        // stub's info into the alloc, so that the alloc is a
                        // more fleshed-out version of the stub.
                        // JobVersion is not in the Allocation, only the stub

                        if allocStub.JobVersion != *alloc.Job.Version {
                            log.Errorf("fack. allocStub.JobVersion (%d) != *alloc.Job.Version (%d)", allocStub.JobVersion, *alloc.Job.Version)
                        }

                        // these seem most likely to change between the stub and
                        // the retrieved allocation
                        alloc.DesiredStatus      = allocStub.DesiredStatus
                        alloc.DesiredDescription = allocStub.DesiredDescription
                        alloc.ClientStatus       = allocStub.ClientStatus
                        alloc.ClientDescription  = allocStub.ClientDescription
                        alloc.TaskStates         = allocStub.TaskStates
                        alloc.DeploymentStatus   = allocStub.DeploymentStatus
                        alloc.RescheduleTracker  = allocStub.RescheduleTracker
                        alloc.CreateIndex        = allocStub.CreateIndex
                        alloc.ModifyIndex        = allocStub.ModifyIndex
                        alloc.CreateTime         = allocStub.CreateTime
                        alloc.ModifyTime         = allocStub.ModifyTime

                        allocEventChan <- AllocEvent{
                            ts,
                            queryMeta.LastIndex,
                            alloc,
                        }
                    }

                    lastTaskEventTime := allocTaskEventTimes[allocId]
                    for taskName, taskState := range allocStub.TaskStates {
                        // assumption is taskState.Events is always ordered
                        for _, taskEvent := range taskState.Events {
                            if taskEvent.Time > lastTaskEventTime {
                                // new TaskEvent

                                // emit only if the allocation was updated
                                if allocationUpdated {
                                    taskStateEventChan <- TaskStateEvent{
                                        Timestamp: time.Unix(0, taskEvent.Time),
                                        WaitIndex: queryMeta.LastIndex,

                                        TaskState: TaskState{
                                            JobID:        allocStub.JobID,
                                            AllocID:      allocId,
                                            AllocName:    allocStub.Name,
                                            TaskGroup:    allocStub.TaskGroup,
                                            EvalID:       allocStub.EvalID,
                                            NodeID:       allocStub.NodeID,
                                            DeploymentID: deployId,

                                            Task:   taskName,
                                            State:  taskState.State,
                                            Failed: taskState.Failed,

                                            TaskEvent: taskEvent,
                                        },
                                    }
                                }

                                // store the timestamp of the most recent task
                                // event
                                allocTaskEventTimes[allocId] = taskEvent.Time
                            }
                        }
                    }
                }

                // prune allocs
                for allocId, _ := range allocTaskEventTimes {
                    if _, ok := extantAllocs[allocId]; ! ok {
                        log.Infof("allocation %s has been deleted", allocId)

                        delete(allocTaskEventTimes, allocId)
                    }
                }
            }

            queryOpts.WaitIndex = queryMeta.LastIndex
        }
    }()

    return allocEventChan, taskStateEventChan
}
