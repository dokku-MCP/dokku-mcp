# Licensing Strategy

## Current Status

**Current License**: Apache License 2.0  
**Status**: Dual-project strategy with separate SaaS component

## Goals

This project aims to balance:
- **Maximum community contribution** for the core
- **Long-term economic viability** through SaaS offerings
- **Strategic flexibility** as the project evolves

## Dual-Project Architecture

### 1. dokku-mcp (Apache 2.0) - Core Open Source
```
✅ Maximum adoption
✅ Free contributions
✅ Compatible with Go ecosystem
✅ Self-hosted deployments
✅ Community-driven development
```

**Features**:
- Core MCP server functionality
- Dokku integration
- CLI tools and basic API
- Essential monitoring
- Single-tenant architecture

### 2. dokku-mcp-cloud (BSL) - SaaS Platform
```
✅ Commercial control
✅ Revenue generation
✅ Enterprise features
✅ Managed infrastructure
✅ Professional support
```

**Features**:
- Multi-tenant web dashboard
- Team management & RBAC
- Advanced monitoring & alerts
- Billing & subscription management
- Enterprise integrations (SSO, audit)
- 24/7 support

## Legal Compatibility

### ✅ **Perfectly Legal**
- **Apache 2.0 permits commercial use**: You can use dokku-mcp in your SaaS
- **No restrictions**: Apache 2.0 imposes no constraints on your business model
- **Separate codebases**: dokku-mcp-cloud is your intellectual property

### ✅ **Apache 2.0 Obligations**
```
✅ Keep copyright notices from dokku-mcp
✅ Include Apache 2.0 license in attributions
❌ No obligation to share dokku-mcp-cloud source
❌ No restrictions on your pricing model
```

## Reference Examples

Projects successfully using this dual approach:
- **Grafana** (Apache 2.0) + **Grafana Cloud** (proprietary)
- **Supabase** (Apache 2.0) + **Supabase Cloud** (proprietary)
- **PostHog** (MIT) + **PostHog Cloud** (proprietary)
- **Plausible** (AGPL) + **Plausible.io** (hosted service)

## Project Structure

```
github.com/your-org/
├── dokku-mcp/              # Apache 2.0 - Core server
│   ├── internal/domain/
│   ├── internal/application/
│   ├── cmd/server/
│   └── LICENSE (Apache 2.0)
│
└── dokku-mcp-cloud/        # BSL - SaaS platform
    ├── web/                # React/Vue dashboard
    ├── api/                # Multi-tenant API
    ├── billing/            # Stripe integration
    ├── deployment/         # Kubernetes infrastructure
    └── LICENSE (BSL)
```

## Target Users

### dokku-mcp (Open Source)
- Individual developers
- Small teams
- Self-hosting enthusiasts
- Open source projects
- Educational use

### dokku-mcp-cloud (SaaS)
- Growing teams
- Enterprises
- Clients wanting managed service
- Organizations needing compliance
- Companies requiring support SLAs

## Go-to-Market Strategy

### Phase 1: Build Core (Apache 2.0)
1. Develop dokku-mcp with all essential features
2. Build strong community
3. Validate product-market fit
4. Document and stabilize API

### Phase 2: Launch SaaS (BSL)
1. Create dokku-mcp-cloud with business layer
2. Multi-tenant architecture
3. Web interface and billing
4. Beta launch then production

### Phase 3: Scale Both
1. Community contributions to core
2. Enterprise sales for SaaS
3. Ecosystem of integrations
4. Professional services

## Decision Criteria

This dual approach is optimal because:
- [ ] Core functionality benefits from community contributions
- [ ] SaaS layer provides sustainable revenue
- [ ] No license conflicts or legal issues
- [ ] Proven model by successful companies
- [ ] Maximum flexibility for future growth

## Communication Strategy

**For Open Source Community**:
- Clear separation of core vs. SaaS features
- Transparent roadmap for both projects
- Community input on core development
- Open governance for dokku-mcp

**For Commercial Customers**:
- Value proposition of managed service
- Professional support and SLAs
- Enterprise features not in core
- Migration path from self-hosted

## Implementation Timeline

### Q1: Core Development
- [ ] Complete dokku-mcp basic functionality
- [ ] Establish contributor guidelines
- [ ] Create comprehensive documentation
- [ ] Build initial community

### Q2: SaaS Planning  
- [ ] Design multi-tenant architecture
- [ ] Create dokku-mcp-cloud repository
- [ ] Plan billing and subscription model
- [ ] Design web dashboard

### Q3: SaaS Development
- [ ] Implement multi-tenant backend
- [ ] Build web interface
- [ ] Integrate billing system
- [ ] Set up hosted infrastructure

### Q4: Launch
- [ ] Beta testing with select customers
- [ ] Public launch of SaaS
- [ ] Marketing and sales campaigns
- [ ] Community growth initiatives

## Contact

For licensing questions:
- **Issues**: Open a GitHub issue
- **Email**: [your-email] for commercial inquiries
- **Discussions**: GitHub Discussions for community topics

---

*This strategy provides maximum flexibility while building both community and business value.* 