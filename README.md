# Who watches the watcher?

This is a simple service that watches [Nomad](https://nomadproject.io)'s nodes, jobs, deployments, evaluations, allocations, and task states, and writes the events to a file.  The intention is that they'll be shipped to a central log collection system so users and operators can get insight into what Nomad is doing.

## usage

    nomad-watcher --events-file=nomad_events.json

## events

The events all have the same basic structure:

* `@timestamp` — Timestamp in RFC 3339 format.
* `wait_index` — The value of the `X-Nomad-Index` header in the response to the [blocking query](https://www.nomadproject.io/api/index.html#blocking-queries)
* `type` — The type of the event; one of `alloc`, `task_state`, `eval`, `job`, `node`, or `deploy`.
* `AllocationID` — The event's allocation ID, if applicable.
* `EvaluationID` — The event's evaluation ID, if applicable.
* `JobID` — The event's job ID, if applicable.
* `NodeID` — The event's node ID, if applicable.
* `DeploymentID` — The event's deployment ID, if applicable.
* `event` — The details of the triggered event. This is the serialized [Nomad API struct](https://godoc.org/github.com/hashicorp/nomad/api).

# nomad-tail

I found myself wanting to see the events scrolling by in a console window, so I shaved a yak and created `nomad-tail`.

## usage

    nomad-tail

## sample output

Everything's colorized, but you'll have to take my word for it:

    <eval  > E[ddfe9d8a] system-date job-deregister pending      D[        ] N[        ] next: E[        ] prev: E[        ] block: E[        ]
    <job   > system-date v001 system 050 dead
    <eval  > E[ddfe9d8a] system-date job-deregister complete     D[        ] N[        ] next: E[        ] prev: E[        ] block: E[        ]
    <alloc > A[b618a2de] system-date v000 E[2b880115] N[47834f73] stop alloc not needed due to job update running
    <task  > A[b618a2de] system-date.date[0] dead          Killing DisplayMessage=Sent interrupt. Waiting 5s before force killing
    <task  > A[b618a2de] system-date.date[0] dead          Killed DisplayMessage=Task successfully killed
    <alloc > A[b618a2de] system-date v000 E[2b880115] N[47834f73] stop alloc not needed due to job update complete

No timestamps are shown to reduce clutter. iTerm2's "Show Timestamps" feature works well here.

# building

You need a Go development environment.  Modules are used, so Go >= 1.11 is required.

    make
  
Binaries are placed into `stage/`.
