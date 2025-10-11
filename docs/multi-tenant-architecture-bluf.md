# Dokku-MCP Multi-Tenant SaaS Architecture - BLUF

## Executive Summary

**Bottom Line Up Front:** Transform dokku-mcp into a secure, multi-tenant SaaS offering by leveraging the mcp-go library's **native `WithSSEContextFunc`** authentication mechanism, implementing a **Domain-Driven Design authentication bounded context**, and using **HashiCorp Vault for dynamic tenant credential management**—all without modifying the Apache-licensed core runtime.

---

## 🎯 Strategic Recommendation

### The Native Approach

Instead of building external API gateways or proxy layers, **integrate authentication directly into dokku-mcp** using:

1. **MCP Protocol-Native Authentication**
   - Use `server.WithSSEContextFunc()` from mcp-go library
   - Inject tenant context from SSE connection query parameters
   - No protocol violations, no external proxies needed

2. **DDD Architecture with Authentication Bounded Context**
   - Separate authentication domain from Dokku management domain
   - Clean interfaces between layers (Domain → Application → Infrastructure)
   - Maintainable, testable, and extensible

3. **Dynamic Secret Management**
   - HashiCorp Vault for per-tenant SSH credentials
   - Short-lived secrets (1-24 hour TTL)
   - Automatic rotation without service interruption

---

## 💼 Business Value Proposition

### Revenue Model Enabled

| Tier | Features | Pricing Model |
|------|----------|---------------|
| **Free** | Read-only access, limited apps | $0/month |
| **Pro** | Full app management, 10 apps | $49/month |
| **Enterprise** | Unlimited apps, dedicated Dokku instance, SLA | $499/month |

### Key Differentiators

- **Zero Infrastructure Overhead**: Customers don't need Dokku expertise
- **Multi-Tenant Security**: Complete isolation between tenants
- **Observable**: Full audit trails for compliance (SOC 2, GDPR)
- **API-First**: Integrate with CI/CD pipelines, ChatOps, LLM agents

---

## 🏗️ Technical Architecture

### High-Level Component View

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Applications                       │
│  • Web Dashboard                                             │
│  • CLI Tools                                                 │
│  • CI/CD Integrations (GitHub Actions, GitLab CI)          │
│  • LLM Agents (Claude, GPT-4)                               │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ HTTPS + OAuth 2.0
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Dokku-MCP SSE Server (Enhanced)                 │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  MCP Transport Layer (mcp-go)                          │ │
│  │  • SSE endpoint with WithSSEContextFunc                │ │
│  │  • Authenticates token from query param                │ │
│  │  • Injects tenant context into Go context.Context      │ │
│  └────────────────────────────────────────────────────────┘ │
│                              │                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Authentication Bounded Context (NEW)                  │ │
│  │  • Validates JWT tokens                                │ │
│  │  • Enforces permission model                           │ │
│  │  • Retrieves tenant-specific credentials from Vault    │ │
│  └────────────────────────────────────────────────────────┘ │
│                              │                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Dokku Management Domain (Existing)                    │ │
│  │  • App, Deployment, Service plugins                    │ │
│  │  • Uses tenant context for authorization checks        │ │
│  └────────────────────────────────────────────────────────┘ │
│                              │                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Dokku Client (Context-Aware)                          │ │
│  │  • Extracts tenant SSH config from context             │ │
│  │  • Connects to tenant's Dokku instance                 │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
    ┌──────────┐        ┌──────────┐        ┌──────────┐
    │  Vault   │        │ Postgres │        │  Dokku   │
    │(Secrets) │        │(Tenants) │        │Instances │
    └──────────┘        └──────────┘        └──────────┘
```

---

## 🔐 Security Architecture

### Three-Layer Security Model

**Layer 1: Client Authentication**
- OAuth 2.0 Client Credentials Grant
- JWT tokens with 15-minute expiry
- Refresh tokens for long-lived sessions

**Layer 2: Authorization**
- Role-Based Access Control (RBAC)
- Permission model: `apps:read`, `apps:deploy`, `apps:destroy`, etc.
- Enforced at every MCP tool invocation

**Layer 3: Tenant Isolation**
- Dynamic SSH credentials per tenant (Vault-issued)
- Process-level isolation (dedicated pods in Kubernetes)
- Network policies preventing cross-tenant access

### Secret Management Flow

```
1. Tenant provisions account
   → Vault generates unique SSH key pair
   → Private key stored encrypted in Vault
   → Public key added to tenant's Dokku instance

2. Client connects with JWT
   → SSEContextFunc validates JWT
   → Retrieves tenant_id from token
   → Requests SSH credentials from Vault (1-hour TTL)
   → Materializes private key to ephemeral memory mount

3. Tool execution
   → Uses tenant-specific SSH credentials
   → Connects to tenant's Dokku host
   → Executes command securely

4. Credential cleanup
   → Private key deleted from memory after use
   → Vault auto-rotates keys every 24 hours
```

---

## 📊 Deployment Strategy

### Cloud-Native Deployment (Kubernetes)

**Option A: Single Shared dokku-mcp Instance**
- All tenants connect to one scalable deployment
- Context injection provides tenant isolation
- Cost-effective for high tenant density
- Suitable for Free/Pro tiers

**Option B: Per-Tenant dokku-mcp Pods**
- Dedicated pod per enterprise tenant
- Complete process isolation
- Custom resource limits per tenant
- Suitable for Enterprise tier with SLA requirements

**Option C: Hybrid Approach (Recommended)**
- Shared instances for Free/Pro tiers
- Dedicated instances for Enterprise tier
- Dynamic provisioning based on demand

### Infrastructure Requirements

```yaml
# Minimum viable deployment
Components:
  - dokku-mcp deployment: 3 replicas (HA)
  - PostgreSQL: Tenant metadata and auth
  - HashiCorp Vault: Secret management
  - Ingress: TLS termination and routing
  - Prometheus + Grafana: Observability

Estimated Monthly Cost:
  - Kubernetes cluster (GKE/EKS/AKS): $150-300
  - Managed PostgreSQL: $50-100
  - Vault (self-hosted): $0
  - Load balancer + bandwidth: $50-150
  Total: ~$250-550/month base infrastructure
```

---

## 📈 Implementation Roadmap

### Phase 1: Foundation (4-6 weeks)
**Goal:** Basic multi-tenant authentication working

- Week 1-2: Implement Authentication domain (entities, services, interfaces)
- Week 3-4: Build SSEContextAuthenticator with mcp-go integration
- Week 5: Set up Vault and implement VaultSecretProvider
- Week 6: Testing and documentation

**Deliverables:**
- ✅ JWT-based SSE authentication
- ✅ Per-tenant context injection
- ✅ Vault integration for SSH keys
- ✅ Unit and integration tests

### Phase 2: Authorization & Observability (2-3 weeks)
**Goal:** Production-ready security and monitoring

- Week 7: Implement AuthorizedToolHandler for all MCP tools
- Week 8: Add tenant-scoped logging and metrics
- Week 9: Set up Prometheus dashboards and alerts

**Deliverables:**
- ✅ RBAC enforcement on all operations
- ✅ Comprehensive audit logging
- ✅ Real-time monitoring dashboards

### Phase 3: Production Deployment (2-3 weeks)
**Goal:** Live SaaS service

- Week 10: Kubernetes deployment configuration
- Week 11: CI/CD pipeline setup
- Week 12: Load testing and optimization
- Week 13: Beta launch with select customers

**Deliverables:**
- ✅ Kubernetes manifests and Helm charts
- ✅ Automated deployment pipeline
- ✅ Load-tested for 100+ concurrent tenants
- ✅ Customer onboarding portal

### Phase 4: Scale & Iterate (Ongoing)
**Goal:** Growth and feature expansion

- Metrics-driven optimization
- Customer feedback integration
- Additional Dokku plugin support
- Advanced features (auto-scaling, cost analytics)

---

## 💰 Total Cost of Ownership

### Development Costs

| Phase | Effort | Estimated Cost |
|-------|--------|----------------|
| Authentication implementation | 4-6 weeks | $15,000 - $25,000 |
| Security & authorization | 2-3 weeks | $8,000 - $12,000 |
| Deployment & DevOps | 2-3 weeks | $8,000 - $12,000 |
| Testing & QA | 2 weeks | $6,000 - $8,000 |
| Documentation | 1 week | $3,000 - $4,000 |
| **Total** | **11-15 weeks** | **$40,000 - $61,000** |

### Monthly Operating Costs

| Component | Cost Range |
|-----------|------------|
| Infrastructure (K8s, DB, etc.) | $250 - $550 |
| Monitoring & Logging | $100 - $300 |
| Support & Maintenance (10%) | $200 - $400 |
| **Total per month** | **$550 - $1,250** |

### Break-Even Analysis

```
Assumptions:
- Average customer: $49/month (Pro tier)
- Operating costs: $800/month
- Support time: 2 hours/month per customer

Break-even: ~17 paying customers
Profitable at: 25+ customers ($1,225+ MRR)
```

---

## ⚖️ Licensing Strategy

### Core Runtime (Apache 2.0)
- `dokku-mcp` core remains open source
- Community contributions welcome
- No changes to license

### SaaS Wrapper (Proprietary)
- Authentication bounded context: Proprietary
- Multi-tenant infrastructure: Proprietary
- Management dashboard: Proprietary
- Allows commercial SaaS without relicensing core

### Compliance
- Apache 2.0 allows commercial use
- Must include Apache license notice
- Can add proprietary components on top
- Clear separation maintained

---

## 🎯 Success Metrics

### Technical KPIs

- **Uptime**: 99.9% availability (SLA for Enterprise)
- **Performance**: <200ms p95 latency for MCP tool calls
- **Security**: Zero security incidents, SOC 2 compliance
- **Scalability**: Support 1000+ concurrent tenants

### Business KPIs

- **Customer Acquisition**: 50 beta users in Month 1
- **Conversion Rate**: 20% free → paid conversion
- **Churn**: <5% monthly churn rate
- **NPS**: Net Promoter Score >40

---

## 🚀 Go-to-Market Strategy

### Target Customers

**Primary:**
- Indie developers and small teams using Dokku
- DevOps consultancies managing multiple Dokku deployments
- SaaS companies wanting PaaS without Heroku costs

**Secondary:**
- LLM application developers needing infrastructure automation
- Education institutions teaching cloud deployment
- Agencies building client applications

### Value Propositions

1. **For Developers**: "Deploy like Heroku, control like AWS, pay like DigitalOcean"
2. **For Agencies**: "Manage all client Dokku instances from one dashboard"
3. **For LLM Apps**: "Give your AI agent infrastructure management superpowers"

---

## ⚠️ Risk Assessment

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Security breach** | CRITICAL | LOW | Multi-layer security, regular audits, bug bounty |
| **Vault downtime** | HIGH | MEDIUM | Credential caching, fallback mechanisms |
| **Dokku compatibility** | MEDIUM | MEDIUM | Comprehensive testing, version matrix |
| **Competitor entry** | MEDIUM | HIGH | First-mover advantage, open-source foundation |
| **Regulatory compliance** | HIGH | MEDIUM | SOC 2 certification, GDPR compliance |

---

## 📚 Next Steps

### Immediate Actions (This Week)

1. **Technical Validation**
   - [ ] Prototype SSEContextFunc integration
   - [ ] Test JWT validation performance
   - [ ] Verify Vault SSH secret engine capabilities

2. **Business Validation**
   - [ ] Survey 20 potential customers
   - [ ] Validate pricing model
   - [ ] Assess competitive landscape

3. **Resource Planning**
   - [ ] Identify development team (2-3 engineers)
   - [ ] Budget approval ($50k-70k for Phase 1-3)
   - [ ] Timeline commitment (3-4 months to launch)

### Decision Point

**Go/No-Go Criteria:**
- ✅ Technical feasibility validated (mcp-go supports needed features)
- ⏳ Customer interest confirmed (>30 signups for beta)
- ⏳ Budget approved
- ⏳ Team assigned

---

## 📞 Stakeholder Communication

### For Engineering Leadership
"We can build a secure, multi-tenant SaaS on top of dokku-mcp in 3-4 months using the mcp-go library's native authentication hooks. The architecture follows clean DDD principles, maintains Apache licensing compliance, and requires no modifications to the open-source core."

### For Product/Business
"There's a market gap for affordable, Dokku-as-a-Service targeting developers who find Heroku too expensive and AWS too complex. We can capture this market with a $40-60k initial investment and break even at ~20 paying customers."

### For Investors
"Dokku has 29k GitHub stars but no official managed service. We're positioning to become the 'Heroku for Dokku' with better pricing and LLM-integration capabilities. TAM: $500M (subset of $50B PaaS market). SAM: $50M (Dokku user base). SOM: $5M (achievable in Year 1)."

---

## 🏁 Conclusion

The MCP-native authentication approach using `WithSSEContextFunc` provides a **technically elegant and commercially viable path** to transform dokku-mcp into a multi-tenant SaaS offering. By leveraging:

- **Built-in mcp-go features** (no protocol hacks)
- **Clean DDD architecture** (maintainable and extensible)
- **Modern secret management** (Vault for dynamic credentials)
- **Cloud-native deployment** (Kubernetes-ready)

We can deliver a **production-ready service in 3-4 months** with a **reasonable investment** and **strong market positioning**.

**Recommendation: Proceed with Phase 1 implementation.**

