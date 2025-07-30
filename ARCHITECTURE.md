# StormDB Architecture Diagram

```
                              🌟 StormDB Architecture 🌟
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                                    USER LAYER                                │
    ├─────────────────────────────────────────────────────────────────────────────┤
    │  📋 Config Files        🎯 CLI Interface       📊 Demo Scripts              │
    │  ┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐           │
    │  │ config_*.yaml   │   │ ./stormdb       │   │ demo_*.sh       │           │
    │  │ • Database      │   │ --config        │   │ • Interactive   │           │
    │  │ • Workload      │   │ --setup         │   │ • Examples      │           │
    │  │ • Plugins       │   │ --rebuild       │   │ • Tutorials     │           │
    │  │ • Metrics       │   │ --pg-stats      │   │                 │           │
    │  └─────────────────┘   └─────────────────┘   └─────────────────┘           │
    └─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                                  CORE ENGINE                                 │
    ├─────────────────────────────────────────────────────────────────────────────┤
    │  🔧 Config Manager    📈 Metrics Engine    🗄️  Database Layer              │
    │  ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐               │
    │  │ • YAML Parse    │ │ • Latency       │ │ • PostgreSQL    │               │
    │  │ • Validation    │ │ • Throughput    │ │ • Connection    │               │
    │  │ • Plugin Config │ │ • Percentiles   │ │   Pool          │               │
    │  └─────────────────┘ └─────────────────┘ │ • pg_stats      │               │
    │                                          │ • Monitoring    │               │
    │  🚦 Signal Handler   ⚡ Worker Pool      └─────────────────┘               │
    │  ┌─────────────────┐ ┌─────────────────┐                                   │
    │  │ • Graceful      │ │ • Concurrency   │                                   │
    │  │   Shutdown      │ │ • Load          │                                   │
    │  │ • Interrupts    │ │   Distribution  │                                   │
    │  └─────────────────┘ └─────────────────┘                                   │
    └─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                              WORKLOAD FACTORY                               │
    ├─────────────────────────────────────────────────────────────────────────────┤
    │              🏭 Dynamic Workload Discovery & Management                     │
    │                                                                             │
    │  ┌─────────────────┐        ┌─────────────────┐        ┌─────────────────┐ │
    │  │   Built-in      │        │     Plugin      │        │   Lifecycle     │ │
    │  │   Registry      │        │   Discovery     │        │   Management    │ │
    │  │                 │        │                 │        │                 │ │
    │  │ • TPCC          │◄──────►│ • Auto-scan     │◄──────►│ • Initialize    │ │
    │  │ • Simple        │        │ • Load .so/.dll │        │ • Create        │ │
    │  │ • Connection    │        │ • Metadata      │        │ • Cleanup       │ │
    │  │   Overhead      │        │ • Validation    │        │ • Error Handle  │ │
    │  └─────────────────┘        └─────────────────┘        └─────────────────┘ │
    └─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                              PLUGIN ECOSYSTEM                               │
    ├─────────────────────────────────────────────────────────────────────────────┤
    │                            🔌 Dynamically Loaded                           │
    │                                                                             │
    │ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────┐ │
    │ │  🎬 IMDB        │ │  🔍 Vector      │ │  🛒 E-commerce  │ │ 🛍️ Ecom Basic│ │
    │ │  Plugin         │ │  Plugin         │ │  Plugin         │ │  Plugin     │ │ 
    │ │                 │ │                 │ │                 │ │             │ │
    │ │• Movie DB       │ │• pgvector       │ │• Orders         │ │• Enterprise │ │
    │ │• Complex        │ │• Similarity     │ │• Inventory      │ │• OLTP/OLAP  │ │
    │ │  Queries        │ │  Search         │ │• Analytics      │ │• Business   │ │
    │ │• Analytics      │ │• High-dim       │ │• Vendors        │ │  Logic      │ │
    │ │                 │ │  Vectors        │ │• Reviews        │ │             │ │
    │ │📁 imdb_plugin   │ │📁 vector_plugin │ │📁 ecommerce_    │ │📁 ecommerce_│ │
    │ │                │ │                │ │   plugin       │ │   basic_    │ │
    │ │   .so           │ │   .so           │ │   plugin.so     │ │   plugin.so │ │
    │ └─────────────────┘ └─────────────────┘ └─────────────────┘ └─────────────┘ │
    │                                                                             │
    │ ┌─────────────────────────────────────────────────────────────────────────┐ │
    │ │                          🛠️  Custom Plugins                            │ │
    │ │ ┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐           │ │
    │ │ │   Community     │ │   Enterprise    │ │    Future       │           │ │
    │ │ │   Plugins       │ │   Extensions    │ │   Extensions    │           │ │
    │ │ │                 │ │                 │ │                 │           │ │
    │ │ │ • User-contrib  │ │ • Custom        │ │ • AI/ML         │           │ │
    │ │ │ • Open Source   │ │ • Proprietary   │ │ • Specialized   │           │ │
    │ │ │ • Domain        │ │ • Industry      │ │ • Advanced      │           │ │
    │ │ │   Specific      │ │   Specific      │ │   Analytics     │           │ │
    │ │ └─────────────────┘ └─────────────────┘ └─────────────────┘           │ │
    │ └─────────────────────────────────────────────────────────────────────────┘ │
    └─────────────────────────────────────────────────────────────────────────────┘
                                          │
                                          ▼
    ┌─────────────────────────────────────────────────────────────────────────────┐
    │                              OUTPUT LAYER                                   │
    ├─────────────────────────────────────────────────────────────────────────────┤
    │   📊 Metrics & Analysis     🔍 Monitoring        📈 Reports                │
    │   ┌─────────────────┐      ┌─────────────────┐   ┌─────────────────┐       │
    │   │ • TPS/QPS       │      │ • Real-time     │   │ • Summary       │       │
    │   │ • Latency       │      │ • pg_stats      │   │ • Detailed      │       │
    │   │ • Percentiles   │      │ • Buffer Cache  │   │ • Worker        │       │
    │   │ • Histograms    │      │ • WAL Activity  │   │   Breakdown     │       │
    │   │ • Error Rates   │      │ • Connections   │   │ • Trends        │       │
    │   └─────────────────┘      └─────────────────┘   └─────────────────┘       │
    └─────────────────────────────────────────────────────────────────────────────┘

  📋 Key Features:
  • Modular plugin architecture with dynamic loading
  • Built-in workloads (TPCC, Simple, Connection Overhead) 
  • Plugin workloads (IMDB, Vector, E-commerce, E-commerce Basic)
  • Comprehensive PostgreSQL monitoring and metrics
  • Easy extensibility through custom plugin development
  • Production-ready performance testing capabilities
```

## Build Process Flow

```
    Developer Workflow:
    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
    │   📝 Code       │ ──► │   🔨 Build      │ ──► │   🚀 Deploy     │
    │   • main.go     │    │   make build    │    │   ./stormdb     │
    │   • plugins/    │    │   make plugins  │    │   --config      │
    │   • configs/    │    │   make build-all│    │   config.yaml   │
    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                   │
                                   ▼
    Plugin Build Chain:
    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
    │   Plugin Src    │ ──► │   Compile       │ ──► │   Load & Run    │
    │   *.go files    │    │   .so/.dll      │    │   Runtime       │
    │   package main  │    │   -buildmode=   │    │   Discovery     │
    │                 │    │   plugin        │    │                 │
    └─────────────────┘    └─────────────────┘    └─────────────────┘
```
