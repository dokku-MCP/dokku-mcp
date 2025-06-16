## **MCP Server for Dokku: Product Specification**

### 1. Introduction

#### 1.1. Executive Summary
This document outlines the product specification for a **dynamically modular** and **workflow-driven** Model Context Protocol (MCP) server that acts as an intelligent, conversational interface to a Dokku installation. Dokku is a powerful, Docker-based Platform as a Service (PaaS) that simplifies application deployment and management.

This MCP server exposes Dokku's functionality through a standardized, secure protocol. Its architecture is built around a **dynamic plugin registry** and an **abstracted workflow engine**. This design allows the server to hot-reload its capabilities in real-time as Dokku plugins are installed or removed, without requiring a server restart. Furthermore, its workflow system is designed to be source-agnostic, supporting definitions from local YAML files, databases, or remote APIs, ensuring maximum flexibility for developers and administrators.

#### 1.2. Goals & Objectives
*   **Enable Conversational Management:** Allow users to manage their Dokku server using natural language commands via an LLM.
*   **Codify Operational Knowledge:** Provide a declarative system for defining multi-step operational **Workflows** that can be sourced from various backends (e.g., files, database).
*   **Simplify Complex Operations:** Abstract intricate procedures into single, executable workflows.
*   **Provide Full Observability:** Expose all of Dokku's state and configuration as read-only MCP Resources.
*   **Ensure Secure & Controlled Actions:** Expose all state-changing Dokku commands as MCP Tools.
*   **Embrace Dynamic Modularity:** Design the server to automatically detect and adapt to changes in the underlying Dokku installation (e.g., new plugins) without restarts.

### 2. Server Architecture

#### 2.1. Core Design: Dynamic Registry and Abstracted Engines
The server's architecture is centered around a **dynamic plugin registry** and a **pluggable workflow provider system**.

*   **Dynamic Plugin Registry:** The server maintains an in-memory registry of all available `FeaturePlugin` implementations. Instead of a one-time load, the server periodically (or via a trigger) re-scans the Dokku installation to determine which plugins should be active. This allows the server to hot-load and unload functionality as plugins are added or removed from Dokku.
*   **Pluggable Workflow Provider:** The workflow engine is designed to be source-agnostic. It interacts with a `WorkflowProvider` interface, which is responsible for fetching and providing workflow definitions. The initial implementation will be a `YAMLFileProvider` that reads from the local filesystem, but this can be swapped out for other providers (e.g., `DatabaseProvider`, `ApiProvider`) without changing the core engine.

#### 2.2. Security & Transport
*   The server runs as a user with `dokku` privileges and enforces a strict mapping from tools to specific `dokku` commands.
*   The primary transport mechanism will be **Stdio**.

---

### 3. MCP Feature Specification: A Dynamic & Modular Approach

#### 3.1. Dynamic Plugin Lifecycle
The server manages the lifecycle of its feature plugins dynamically to reflect the state of the Dokku host.

1.  **Registration:** On startup, the server scans its codebase for all available `FeaturePlugin` implementations and registers them.
2.  **Activation & Synchronization:**
    *   The server establishes a periodic check (e.g., every 60 seconds) or uses a file system watcher on the Dokku plugins directory.
    *   On each check, it executes `dokku plugin:list` to get the list of currently enabled Dokku plugins.
    *   It compares this list with its currently **active** plugins.
    *   **New Plugins:** If a newly enabled Dokku plugin matches a registered but inactive plugin, that plugin is activated.
    *   **Removed Plugins:** If a disabled Dokku plugin corresponds to an active plugin, that plugin is deactivated.
3.  **Capability Reporting:** When a client sends a `get_capabilities` request, the server constructs its response based *only* on the set of currently **active** plugins and workflows. This ensures the advertised capabilities are always up-to-date.

#### 3.2. Core and Optional Feature Plugins
*(Plugin definitions remain the same as in v1.4)*

#### 3.3. Resources and Tools
*(Resource and Tool definitions remain the same as in v1.4, defined within their respective plugins)*

---

### 4. Pluggable Workflow Engine

This system allows developers to define and contribute operational playbooks that can be loaded from various sources.

#### 4.1. Workflow Provider Interface
To decouple the workflow engine from the storage mechanism, a `WorkflowProvider` interface is defined.

```go
// Example Go interface for a workflow provider
type WorkflowProvider interface {
    // GetWorkflows returns a list of all available workflow definitions.
    GetWorkflows() ([]Workflow, error)
}

// Workflow represents a single declarative workflow.
type Workflow struct {
    Name        string
    Description string
    OwnerPlugin string
    Arguments   []mcp.Parameter
    Steps       []WorkflowStep
}
```

#### 4.2. Initial Implementation: `YAMLFileProvider`
The default implementation will be a `YAMLFileProvider` that reads and parses all `*.yaml` files from a local `workflows/` directory.

#### 4.3. Workflow Definition & Step Types
The structure of the workflow definition (name, description, arguments, steps) and the types of steps (`tool_call`, `read_resource`, `prompt`, `conditional`) remain the same as defined in version 1.3. The engine is only concerned with the `Workflow` struct, not how it was loaded.

#### 4.4. Example Workflow File: `workflows/clone_staging.yaml`
*(YAML definition remains the same as in v1.3)*

#### 4.5. Exposing Workflows via MCP
The workflow system is exposed via two core MCP tools, which query the **currently active** workflows from the `WorkflowProvider`.

*   **`workflow.list()`**: Returns a list of all available workflows.
*   **`workflow.run(name, arguments)`**: Executes a specific workflow by name.

### 5. Advanced Features

#### 5.1. Context-Aware Tool and Workflow Filtering
This is now a core, dynamic feature of the server.

**Implementation Logic:**

1.  **Dynamic Plugin Activation:** The server continuously synchronizes its active plugins with the Dokku host's enabled plugins.
2.  **Dynamic Workflow Loading:** The `WorkflowProvider` supplies a list of all potential workflows. The workflow engine filters this list, activating only those whose `owner_plugin` (if specified) corresponds to an active feature plugin.
3.  **Real-time Capability Reporting:** When an MCP client connects, the `get_capabilities` response is generated on-the-fly by:
    *   Aggregating features from all **active plugins**.
    *   Calling `workflow.list()` to get the list of **active workflows**.

This dynamic system ensures that at any point in time, the MCP server presents an accurate and complete picture of the available functionality, adapting automatically to changes in the underlying Dokku environment without needing a restart.

#### 5.2. Developer Experience: Extension Model

*   **Adding a New Feature:** A developer creates a new `FeaturePlugin` file. Once the code is part of the server binary, it will be automatically registered and activated as soon as the corresponding Dokku plugin is enabled on the host.
*   **Adding a New Workflow:**
    *   **File-based:** A developer simply adds a new YAML file to the `workflows/` directory. The `YAMLFileProvider` will automatically pick it up.
    *   **Database-based:** A developer could implement a `DatabaseProvider` that reads workflow definitions from a database table. The core server logic would not need to change.