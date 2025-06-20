package app

// ApplicationPromptTemplates contains prompt templates for Dokku application analysis
// These templates encode business knowledge about diagnostics and optimization
type ApplicationPromptTemplates struct{}

// PromptTemplate represents a prompt template with its business metadata
type PromptTemplate struct {
	Name         string
	Description  string
	Template     string
	RequiredArgs []string
}

// GetDiagnosticPrompt returns the template for diagnosing an application
// Contains business knowledge about critical aspects to analyze
func (p *ApplicationPromptTemplates) GetDiagnosticPrompt() PromptTemplate {
	return PromptTemplate{
		Name:        "diagnose_application",
		Description: "Analyze potential issues of a Dokku application",
		Template: `Please diagnose the Dokku application "%s".

Analyze the following aspects according to Dokku best practices:

🔍 **Application State**
1. Process status (web, worker, cron)
2. Container health and restarts
3. Recent error logs
4. Performance metrics

⚙️ **Configuration**
1. Critical environment variables
2. Resource configuration (CPU/memory)
3. Scaling parameters
4. Healthcheck configuration

🚀 **Deployment**
1. Recent deployment history
2. Buildpack issues
3. Deployment duration
4. Any rollbacks

🔒 **Security**
1. SSL/TLS configuration
2. Exposed sensitive variables
3. Permissions and access
4. Known vulnerabilities

🌐 **Infrastructure**
1. Domain configuration
2. Proxy and load balancing
3. Database and linked services
4. Storage and volumes

Use the available tools to retrieve this information and provide a structured report with prioritized recommendations.`,
		RequiredArgs: []string{"app_name"},
	}
}

// GetOptimizationPrompt returns the template for optimizing an application
// Contains business knowledge about recommended optimizations
func (p *ApplicationPromptTemplates) GetOptimizationPrompt() PromptTemplate {
	return PromptTemplate{
		Name:        "optimize_application",
		Description: "Generate optimization recommendations for a Dokku application",
		Template: `Please analyze the Dokku application "%s" and provide optimization recommendations based on best practices.

📊 **Performance and Scaling**
1. Load metrics analysis
2. Optimization of the number of instances
3. CPU/memory resource configuration
4. Automatic scaling strategies
5. Request performance and response times

🏗️ **Architecture and Configuration**
1. Buildpack choice and configuration
2. Optimization of environment variables
3. Healthcheck configuration
4. Dependency and cache management
5. Process structure (web/worker/cron)

🌐 **Infrastructure and Network**
1. Domain and SSL configuration
2. Proxy and load balancing optimization
3. CDN and static cache
4. Compression and asset optimization
5. DNS configuration and network performance

💾 **Data and Persistence**
1. Database optimization
2. Cache strategies (Redis, Memcached)
3. Volume and storage management
4. Backup and recovery strategies
5. Migration and query performance

🔒 **Security and Monitoring**
1. Improved SSL/TLS configuration
2. Monitoring and alerts
3. Logs and observability
4. Secrets and sensitive variable management
5. Security audit and vulnerabilities

💰 **Cost and Efficiency**
1. Resource usage optimization
2. Infrastructure cost reduction
3. Energy efficiency
4. Automation of repetitive tasks

For each area, provide:
- Specific and actionable recommendations
- Priority (Critical/Important/Desirable)
- Estimated impact on performance
- Implementation complexity
- Specific Dokku commands if applicable`,
		RequiredArgs: []string{"app_name"},
	}
}

// GetAllPromptTemplates returns all available prompt templates
func (p *ApplicationPromptTemplates) GetAllPromptTemplates() []PromptTemplate {
	return []PromptTemplate{
		p.GetDiagnosticPrompt(),
		p.GetOptimizationPrompt(),
	}
}

// NewApplicationPromptTemplates creates a new instance of the templates
func NewApplicationPromptTemplates() *ApplicationPromptTemplates {
	return &ApplicationPromptTemplates{}
}
