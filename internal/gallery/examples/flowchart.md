---
title: Flowchart
description: Steps of a process as boxes, joined by arrows that show where each step leads.
beta: false
featured: 2
---

## Checking out an order

```mermaid
flowchart TD
    cart([Customer opens the cart]) --> stock{All in stock?}
    stock -- No --> backorder[Offer a backorder]
    stock -- Yes --> address[Confirm the address]
    backorder --> address
    address --> pay[Charge the card]
    pay --> paid{Approved?}
    paid -- No --> retry[Ask for another card]
    retry --> pay
    paid -- Yes --> order[(Save the order)]
    order --> email[Email a receipt]
    order --> warehouse[Queue for packing]
    email --> done([Order placed])
    warehouse --> done
```

## From pull request to production

```mermaid
flowchart LR
    pr[Pull request] --> ci
    subgraph ci [CI]
        direction TB
        lint[Lint] --> test[Test]
        test --> build[Build image]
    end
    ci --> staging{Staging healthy?}
    staging -- Yes --> canary[Canary at 5%]
    staging -- No --> rollback[Roll back]
    canary --> prod[Full rollout]
```

## Shapes in a nightly backup

```mermaid
flowchart TD
    cron((2:00 a.m.)) --> snapshot[[Snapshot PostgreSQL]]
    snapshot --> db[(Snapshot volume)]
    db --> compress[/Compress and encrypt/]
    compress --> upload>Upload to S3]
    upload --> verify{Checksum matches?}
    verify -- Yes --> prune{{Prune old backups}}
    verify -- No --> page[\Page the on-call engineer\]
    prune --> done([Backup complete])
```

## Subgraphs in a tide pool sensor network

```mermaid
flowchart TB
    subgraph pools [Tide pools]
        direction LR
        temp[Temperature] ~~~ salt[Salinity] ~~~ light[Light]
    end
    subgraph shore [Shore station]
        direction LR
        gateway[LoRa gateway] --> buffer[(Local buffer)]
    end
    subgraph cloud [Cloud]
        direction LR
        ingest[Ingest] --> series[(Time series)] --> dash[Dashboard]
    end
    pools -- every 5 minutes --> shore
    shore -- hourly upload --> cloud
```

## Edge styles in a cache lookup

```mermaid
flowchart TD
    req[Page request] ==> cache{In Redis?}
    cache == Hit ==> reply[Return the page]
    cache -- Miss --> db[(PostgreSQL)]
    db --> render[Render the page]
    render --> reply
    render -. write back .-> cache
    cache <-. replicate .-> peer[(Redis replica)]
    reply --o metrics[Latency histogram]
    db --x stale[Skip lagging replicas]
```

## Markdown labels in support triage

```mermaid
flowchart TD
    ticket@{ shape: doc, label: "New ticket" } --> blocked{"`Is the customer **blocked**?`"}
    blocked -- Yes --> urgent["`**Priority 1**
    Page support on call`"]
    blocked -- No --> queue["`Add to the queue
    _answered within one business day_`"]
    urgent --> engineer@{ shape: hex, label: "Escalate to engineering" }
    queue --> reply@{ shape: rounded, label: "Reply with a fix or a workaround" }
    engineer --> reply
    classDef critical stroke-width:3px,font-weight:bold
    class urgent critical
```
