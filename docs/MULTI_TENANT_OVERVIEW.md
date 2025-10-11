# Dokku-MCP Multi-Tenant Architecture - Overview

## 📚 Documentation Index

This directory contains comprehensive documentation for transforming dokku-mcp into a secure, multi-tenant SaaS offering using **MCP-native authentication patterns**.

---

## 🎯 Quick Start - Which Document Do I Need?

### For Business/Product Leaders
👉 **Start here:** [Multi-Tenant Architecture BLUF](./multi-tenant-architecture-bluf.md)
- Executive summary with ROI analysis
- Go-to-market strategy
- Cost estimates and timeline
- Risk assessment

### For Technical Architects
👉 **Start here:** [Multi-Tenant SSE Authentication](./multi-tenant-sse-authentication.md)
- Complete DDD architecture
- Security model with three-layer approach
- Kubernetes deployment patterns
- Vault integration for secret management

### For Developers
👉 **Start here:** [Authentication Implementation Guide](./authentication-implementation-guide.md)
- Step-by-step integration with existing code
- Code examples for each layer
- Testing strategies
- Migration path with zero breaking changes

---

## 🔑 Key Discovery: MCP-Native Authentication

### The Problem
How to add multi-tenant authentication to dokku-mcp's SSE transport without:
- Violating the MCP protocol
- Adding external proxy layers
- Modifying the Apache-licensed core

### The Solution
The `mcp-go` library provides `WithSSEContextFunc` - a **native hook for injecting authentication context** from HTTP requests into the MCP server.

```go
// This is the key to MCP-native authentication
sseServer := server.NewSSEServer(
    mcpServer,
    server.WithSSEContextFunc(authenticator.InjectTenantContext),
)
```

**Why This Matters:**
- ✅ Protocol-compliant (uses built-in mcp-go features)
- ✅ No external dependencies (auth integrated in dokku-mcp)
- ✅ Clean architecture (follows DDD bounded contexts)
- ✅ Testable (domain logic separated from infrastructure)

---

## 🏗️ Architecture at a Glance

```
Client (EventSource)
    ↓ token=JWT_TOKEN (query param)
    ↓
SSEContextFunc (InjectTenantContext)
    ↓ validates JWT
    ↓ retrieves tenant from database
    ↓ gets SSH keys from Vault
    ↓ injects into Go context
    ↓
MCP Protocol Handler
    ↓ extracts tenant context
    ↓ checks permissions
    ↓ uses tenant-specific SSH config
    ↓
Dokku Instance (tenant-specific)
```

### Three-Layer Security Model

| Layer | Authentication Method | Purpose |
|-------|----------------------|---------|
| **Layer 1** | OAuth 2.0 Client Credentials | Client authenticates with JWT |
| **Layer 2** | RBAC Permissions | Tool execution authorization |
| **Layer 3** | Dynamic SSH Keys | Dokku command execution |

---

## 📦 What Gets Built

### New Components (Proprietary SaaS Layer)

```
internal/authentication/          # New bounded context
├── domain/                       # Business logic
│   ├── tenant_identity.go       # Entity: Authenticated tenant
│   ├── access_token.go          # Value Object: JWT token
│   ├── permission.go            # Value Object: RBAC permissions
│   └── authentication_service.go # Domain Service: Auth logic
├── application/                  # Use cases
│   └── authorized_tool_handler.go # Wraps MCP tools with auth
└── infrastructure/               # Integrations
    ├── sse_context_authenticator.go  # KEY: WithSSEContextFunc
    ├── vault_secret_provider.go      # SSH key management
    └── postgres_tenant_repository.go # Tenant storage
```

### Enhanced Components (Existing Code)

- `internal/server/server.go` - Add `WithSSEContextFunc` option
- `internal/dokku-api/client.go` - Read tenant context from Go context
- `internal/server-plugins/*/plugin.go` - Wrap tools with authorization

**Result:** Multi-tenant authentication with ~2,000 lines of new code, zero breaking changes to existing functionality.

---

## 🚀 Implementation Timeline

```
Phase 1: Foundation (4-6 weeks)
├─ Week 1-2: Authentication domain layer
├─ Week 3-4: SSEContextFunc integration
├─ Week 5: Vault + secret management
└─ Week 6: Testing

Phase 2: Authorization (2-3 weeks)
├─ Week 7: RBAC implementation
├─ Week 8: Tool authorization wrappers
└─ Week 9: Observability

Phase 3: Production (2-3 weeks)
├─ Week 10: Kubernetes deployment
├─ Week 11: CI/CD pipeline
└─ Week 12-13: Beta launch

Total: 8-12 weeks to production
```

---

## 💰 Cost Summary

### One-Time Development
- **Total**: $40,000 - $61,000
- **Timeline**: 8-12 weeks
- **Team**: 2-3 engineers

### Monthly Operations
- **Infrastructure**: $250-550
- **Monitoring**: $100-300
- **Support**: $200-400
- **Total**: $550-1,250/month

### Break-Even
- **Customers needed**: ~17 at $49/month
- **Profitable at**: 25+ customers
- **MRR target**: $1,225+

---

## 🎯 Business Model

### Pricing Tiers

| Tier | Price/Month | Features |
|------|-------------|----------|
| **Free** | $0 | 3 apps, read-only |
| **Pro** | $49 | 10 apps, full management |
| **Enterprise** | $499 | Unlimited, dedicated instance, SLA |

### Target Market
- **Primary**: Indie devs & small teams using Dokku
- **Secondary**: DevOps consultancies, LLM app developers
- **TAM**: $500M (subset of $50B PaaS market)

---

## 🔐 Security Highlights

### Tenant Isolation
- Process-level isolation (Kubernetes pods)
- Dynamic SSH keys (1-24 hour TTL)
- Per-tenant credential storage in Vault

### Compliance Ready
- SOC 2 Type II preparation
- GDPR compliance (EU data residency)
- Comprehensive audit logging

### Zero Trust Architecture
- Every tool call requires authentication
- Permission checks at multiple layers
- No shared credentials between tenants

---

## 📊 Technical Advantages

### Why This Approach Wins

| Aspect | Traditional Approach | MCP-Native Approach |
|--------|---------------------|---------------------|
| **Architecture** | External auth proxy | Integrated with WithSSEContextFunc |
| **Protocol** | Custom extensions | Native mcp-go features |
| **Testing** | Complex integration tests | Unit testable domain logic |
| **Deployment** | Multiple services | Single binary |
| **Latency** | Extra network hop | Direct context injection |
| **Maintenance** | Separate proxy codebase | Unified codebase |

### DDD Benefits

- **Bounded Contexts**: Authentication cleanly separated from Dokku management
- **Testability**: Domain logic isolated from infrastructure
- **Flexibility**: Easy to swap Vault for AWS Secrets Manager
- **Type Safety**: Strong typing prevents bugs
- **Maintainability**: Clear layer responsibilities

---

## 🧪 Testing Strategy

### Unit Tests
- Domain entities and value objects
- Authentication service logic
- Permission model

### Integration Tests
- SSE connection with authentication
- JWT validation flow
- Vault secret retrieval

### End-to-End Tests
- Full tenant lifecycle
- Multi-tenant isolation verification
- Load testing (100+ concurrent tenants)

---

## 📈 Success Metrics

### Technical KPIs
- **Uptime**: 99.9% (3 nines)
- **Latency**: <200ms p95 for tool calls
- **Scalability**: 1000+ concurrent tenants

### Business KPIs
- **Beta Users**: 50 in Month 1
- **Conversion**: 20% free → paid
- **Churn**: <5% monthly
- **NPS**: >40

---

## 🗺️ Roadmap

### Q1 2025: Foundation
- [x] Research mcp-go authentication capabilities
- [x] Design DDD architecture
- [ ] Implement authentication bounded context
- [ ] Vault integration

### Q2 2025: Launch
- [ ] RBAC and authorization
- [ ] Kubernetes deployment
- [ ] Beta program (50 users)
- [ ] Public launch

### Q3 2025: Scale
- [ ] 100+ paying customers
- [ ] SOC 2 Type II audit
- [ ] Advanced features (auto-scaling, cost analytics)
- [ ] API v2 with GraphQL

### Q4 2025: Expand
- [ ] Enterprise features
- [ ] White-label offering
- [ ] Partnerships with hosting providers

---

## 🚦 Decision Framework

### Go/No-Go Criteria

| Criteria | Status | Notes |
|----------|--------|-------|
| **Technical Feasibility** | ✅ VALIDATED | mcp-go supports WithSSEContextFunc |
| **Customer Interest** | ⏳ IN PROGRESS | Need 30+ beta signups |
| **Budget Approval** | ⏳ PENDING | $40-60k request |
| **Team Assignment** | ⏳ PENDING | 2-3 engineers needed |
| **Market Validation** | ⏳ IN PROGRESS | Survey 20 Dokku users |

**Next Decision Point:** 2 weeks (after beta signup campaign)

---

## 🎓 Learning Resources

### For Understanding MCP Protocol
- [Model Context Protocol Specification](https://spec.modelcontextprotocol.io/)
- [mcp-go Library Documentation](https://pkg.go.dev/github.com/mark3labs/mcp-go)

### For Understanding DDD
- [Domain-Driven Design by Eric Evans](https://www.domainlanguage.com/ddd/)
- [Implementing Domain-Driven Design by Vaughn Vernon](https://vaughnvernon.com/)

### For Vault Integration
- [HashiCorp Vault SSH Secret Engine](https://developer.hashicorp.com/vault/docs/secrets/ssh)
- [Vault Agent with Kubernetes](https://developer.hashicorp.com/vault/tutorials/kubernetes/agent-kubernetes)

---

## 🤝 Contributing

This multi-tenant architecture is designed to be **built on top of** the open-source dokku-mcp core, which remains Apache 2.0 licensed.

### Open Source (Apache 2.0)
- Core MCP server
- Dokku plugin implementations
- Domain entities for apps, deployments, etc.

### Proprietary SaaS Layer
- Authentication bounded context
- Multi-tenant infrastructure code
- Management dashboard

**Contribution Model:**
- Core improvements benefit everyone
- SaaS layer enables commercial sustainability
- Clear separation maintained

---

## 📞 Questions?

### Technical Questions
- Review the [Implementation Guide](./authentication-implementation-guide.md)
- Check the [Architecture Document](./multi-tenant-sse-authentication.md)

### Business Questions
- Review the [BLUF Document](./multi-tenant-architecture-bluf.md)
- Contact: [Your contact info]

### Getting Started
1. Read the BLUF for business context
2. Review the architecture for technical approach
3. Follow the implementation guide to start coding

---

## 🎉 Summary

**The Path Forward:**

1. **MCP-Native**: Use `WithSSEContextFunc` for authentication
2. **DDD Architecture**: Clean bounded contexts, testable code
3. **Vault Integration**: Dynamic secrets, automatic rotation
4. **Kubernetes-Ready**: Cloud-native deployment
5. **Zero Breaking Changes**: Backward compatible with existing deployments

**Timeline**: 8-12 weeks to production
**Investment**: $40-60k development + $500-1,250/month operations
**Break-Even**: ~20 paying customers at $49/month

**Recommendation**: Proceed with Phase 1 implementation after market validation.

---

*Documentation created: October 2025*
*Status: Research & Planning Phase*
*Next Update: After beta program launch*

