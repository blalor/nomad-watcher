Who watches the watcher?

This is a simple service that watches [Nomad](https://nomadproject.io)'s nodes, jobs, allocations, and evaluations, and writes the events to a file.  The intention is that they'll be shipped to a central log collection system so users and operators can get insight into what Nomad is doing.
