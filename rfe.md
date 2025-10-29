# MLFlow Component Integration for Red Hat OpenShift AI

**Feature Overview:**

MLFlow is the industry-standard open-source platform for managing the complete machine learning lifecycle, including experiment tracking, model versioning, and deployment orchestration. By integrating MLFlow as a first-class, operator-managed component in RHOAI, we enable Operations teams to provide consistent, enterprise-grade MLFlow instances across clusters without manual deployment overhead. This integration delivers on our promise of a complete, turnkey MLOps platform where data scientists can track experiments and manage model artifacts within the same ecosystem that handles their notebooks (Workbenches), model serving (KServe), and model governance (ModelRegistry).

**Business Value:**
* **Operational Efficiency**: Eliminate manual MLFlow installation/configuration - reduce deployment time from hours to minutes
* **Platform Consistency**: Deliver MLFlow with the same lifecycle management guarantees as other RHOAI components (upgrades, RBAC, monitoring)
* **MLOps Completeness**: Close the gap between experiment tracking and production deployment, enabling end-to-end ML workflows within a single platform

**Goals:**

**Primary Goals:**

1. **Enable Self-Service MLFlow Deployment**
   * **Who Benefits**: Cluster admins, RHOAI admins, Ops Engineers
   * **How**: Provide declarative, operator-managed MLFlow deployment via DataScienceCluster CR configuration
   * **Success Metric**: MLFlow deployment reduces from 2-4 hours (manual) to less than 10 minutes (operator-managed)

2. **Ensure Enterprise-Grade Lifecycle Management**
   * **Who Benefits**: Operations teams, Platform Engineers
   * **How**: MLFlow follows the same upgrade, rollback, and health monitoring patterns as existing RHOAI components
   * **Success Metric**: Zero-downtime upgrades; consistent status reporting via DSC CR

3. **Integrate MLFlow Into the RHOAI Ecosystem**
   * **Who Benefits**: Data Scientists, ML Engineers
   * **How**: MLFlow appears in RHOAI Dashboard; integrates with existing auth/RBAC; works seamlessly with Workbenches and ModelRegistry
   * **Success Metric**: 90% of RHOAI users discover MLFlow through Dashboard within first week of enablement

4. **Support Multi-Tenancy and Isolation**
   * **Who Benefits**: Enterprise customers with multiple teams/projects
   * **How**: Enable namespace-scoped MLFlow instances with proper RBAC boundaries
   * **Success Metric**: Support 10+ isolated MLFlow instances per cluster

**What Changes:**

| Today (Manual MLFlow) | Future (Operator-Managed MLFlow) |
|----------------------|----------------------------------|
| Ops teams manually install MLFlow using Helm/YAML | Ops teams set `mlflow.managementState: Managed` in DSC CR |
| Separate lifecycle from other RHOAI components | Unified lifecycle with Dashboard, Workbenches, KServe, etc. |
| Custom monitoring and alerting setup | Integrated with RHOAI monitoring stack (Prometheus) |
| Manual upgrades with potential downtime | Operator-managed upgrades with health checks |
| Disconnected from RHOAI Dashboard | First-class integration with RHOAI Dashboard |

**Non-Functional Goals:**
* **Performance**: MLFlow server startup time less than 2 minutes after CR creation
* **Reliability**: 99.9% uptime for MLFlow control plane
* **Security**: Support OpenShift service mesh integration, TLS by default
* **Observability**: Export MLFlow metrics to RHOAI monitoring stack

**Out of Scope:**

The following capabilities are explicitly **NOT** included in this feature to maintain clear boundaries and avoid scope creep:

* **MLFlow Model Serving** - Model serving is handled by KServe component. MLFlow's role is experiment tracking and model registry; KServe handles production inference endpoints. Users export models from MLFlow and deploy via KServe (standard MLOps pattern).

* **MLFlow Projects Management** - MLFlow Projects (packaging/execution of ML code) overlaps with RHOAI Workbenches and Data Science Pipelines. Including it creates confusion about where code execution happens. Use RHOAI Workbenches for interactive development and DSP for pipeline orchestration.

* **Storage Backend Configuration Options** - To reduce complexity and ensure consistency, MLFlow will use RHOAI's standard persistent storage mechanisms (PVCs). Custom S3/Azure/GCS backends are not operator-managed in the initial release. Advanced users can manually configure storage post-deployment.

* **Custom MLFlow Plugins/Extensions** - Plugin management is a data science concern, not an ops concern. The operator manages the base MLFlow deployment only. Data scientists can install plugins in their MLFlow server namespaces using standard mechanisms.

* **MLFlow Authentication/User Management** - Authentication is handled at the OpenShift platform level (OAuth, RBAC). We will not implement MLFlow's native auth system. Use OpenShift OAuth proxy for authentication (consistent with Dashboard, Workbenches).

* **Migration Tools for Existing MLFlow Instances** - Data migration is customer-specific and high-risk. This feature focuses on greenfield deployments. Provide documentation for manual migration; consider as a future enhancement.

* **Multi-Cluster MLFlow Federation** - Multi-cluster management is not a core operator capability today. Adding it for MLFlow creates architectural inconsistency. Align with broader RHOAI multi-cluster strategy in future releases.

**Requirements:**

### 1. API/CRD Requirements

#### MVP - Must Have

**REQ-API-001**: Define MLFlow Custom Resource (CR)
* Create `/api/components/v1alpha1/mlflow_types.go` with MLFlow CRD schema
* Implement singleton pattern with validation rule: `self.metadata.name == 'default-mlflow'`
* Define `MLFlowCommonSpec` struct for shared configuration between DSC and MLFlow CR
* Define `MLFlowCommonStatus` struct for shared status reporting
* Implement `common.PlatformObject` interface with required methods
* Add kubebuilder markers for cluster-scoped resource, status subresource, and print columns

**REQ-API-002**: Define DSC Integration Types
* Create `DSCMLFlow` struct embedding `common.ManagementSpec` and `MLFlowCommonSpec`
* Create `DSCMLFlowStatus` struct for status reporting in DSC
* Add MLFlow component to DSC `Components` struct in `/api/datasciencecluster/v2/datasciencecluster_types.go`
* Add MLFlow status to `ComponentsStatus` struct

**REQ-API-003**: Namespace Configuration Support
* Add `TrackingNamespace` field to `MLFlowCommonSpec` (type: string, default: "odh-mlflow")
* Validation: RFC 1123 DNS label pattern, max length 63 characters
* Immutability: Add XValidation rule to prevent changes when ManagementState is Managed

**REQ-API-004**: Status Reporting
* Add `TrackingNamespace` field to status (reflects actual deployed namespace)
* Add `URL` field to status (MLFlow Tracking Server URL)
* Implement standard conditions: `Ready` (overall component readiness), `DeploymentsAvailable` (deployment health status)
* Support `ComponentReleaseStatus` for tracking deployed MLFlow releases

#### Post-MVP - Nice to Have

**REQ-API-005**: Storage Configuration - Add optional storage backend configuration (S3, PVC, etc.) and database backend configuration (PostgreSQL, MySQL)

**REQ-API-006**: Authentication/Authorization Configuration - Add optional authentication mechanism selection and RBAC configuration for MLFlow access

### 2. Controller Requirements

#### MVP - Must Have

**REQ-CTRL-001**: Component Handler Registration
* Create `/internal/controller/components/mlflow/mlflow.go`
* Implement `componentHandler` struct with registry methods: GetName, Init, NewCRObject, IsEnabled, UpdateDSCStatus
* Register handler in `init()` function via `cr.Add(&componentHandler{})`

**REQ-CTRL-002**: Component Reconciler
* Create `/internal/controller/components/mlflow/mlflow_controller.go`
* Implement `NewComponentReconciler(ctx context.Context, mgr ctrl.Manager) error`
* Configure reconciler with ownership of standard resources and watches on Namespace and DSCInitialization

**REQ-CTRL-003**: Reconciliation Actions Pipeline
* Create `/internal/controller/components/mlflow/mlflow_controller_actions.go`
* Implement required actions in order: initialize → customizeManifests → releases.NewAction() → configureDependencies → template.NewAction() → kustomize.NewAction() → deploy.NewAction() → deployments.NewAction() → updateStatus → gc.NewAction()

**REQ-CTRL-004**: Manifest Management
* Create `/internal/controller/components/mlflow/mlflow_support.go`
* Define constants: ComponentName, ReadyConditionType, DefaultTrackingNamespace, BaseManifestsSourcePath
* Define image mappings for all MLFlow container images
* Implement `baseManifestInfo()` and `extraManifestInfo()` helpers

**REQ-CTRL-005**: Status Management
* Implement condition management: Ready, DeploymentsAvailable with severity and reasons
* Update status fields: TrackingNamespace, URL
* Sync status from MLFlow CR to DSC status

#### Post-MVP - Nice to Have

**REQ-CTRL-006**: Pre-check Validation - Implement dependency validation (required CRDs, storage availability) and validation action

**REQ-CTRL-007**: Custom Validation Webhooks - Implement admission webhooks for MLFlow CR validation

### 3. Manifest Requirements

#### MVP - Must Have

**REQ-MNFST-001**: Kustomize Manifests Structure
* Create manifest directory structure at `/opt/manifests/mlflow/` with base/ and overlays/odh/ directories
* Include deployment.yaml, service.yaml, params.env, kustomization.yaml

**REQ-MNFST-002**: Core MLFlow Resources
* Deployment for MLFlow tracking server with configurable image, resource limits, health probes, volume mounts
* Service for MLFlow tracking server (ClusterIP on port 5000, optional Route/Ingress)
* ServiceAccount, ConfigMap, Secret for MLFlow configuration and credentials

**REQ-MNFST-003**: RBAC Resources - Role/RoleBinding for in-namespace permissions, ClusterRole/ClusterRoleBinding if cross-namespace access needed

**REQ-MNFST-004**: Parameterization
* Create `params.env` with configurable parameters: TRACKING_NAMESPACE, MLFLOW_IMAGE, MLFLOW_IMAGE_TAG
* Support image override via RELATED_IMAGE environment variables

**REQ-MNFST-005**: Labeling - Apply standard ODH labels to all resources: app.kubernetes.io/name, app.kubernetes.io/component, app.kubernetes.io/part-of, opendatahub.io/component

#### Post-MVP - Nice to Have

**REQ-MNFST-006**: Advanced Storage Configuration - PVC templates, S3 configuration, database StatefulSet

**REQ-MNFST-007**: Observability Resources - ServiceMonitor, Grafana dashboard ConfigMap, logging configuration

### 4. Integration Requirements

#### MVP - Must Have

**REQ-INTG-001**: DSCInitialization Integration - Watch for DSCInitialization changes and respect platform-level settings

**REQ-INTG-002**: Namespace Lifecycle Management - Create tracking namespace automatically when component is enabled; do not delete namespace when component is disabled (data preservation)

**REQ-INTG-003**: ManagementState Transitions - Support Managed state (deploy and maintain) and Removed state (remove resources but preserve namespace)

**REQ-INTG-004**: Garbage Collection - Clean up resources when component is removed; preserve user data in namespace

#### Post-MVP - Nice to Have

**REQ-INTG-005**: Dashboard Integration - Register MLFlow in RHOAI dashboard with navigation link and status display

**REQ-INTG-006**: Workbench Integration - Configure workbench pods with MLFlow tracking URI and client libraries

### 5. Testing Requirements

#### MVP - Must Have

**REQ-TEST-001**: Unit Tests - Test component handler methods, reconciliation actions, status condition logic, manifest parameterization; minimum 70% code coverage for new code

**REQ-TEST-002**: E2E Tests
* Create `/tests/e2e/mlflow_test.go` following established patterns
* Test cases: ValidateComponentEnabled, ValidateOperandsOwnerReferences, ValidateUpdateDeploymentsResources, ValidateComponentReleases, ValidateResourceDeletionRecovery, ValidateComponentDisabled

**REQ-TEST-003**: Integration Tests - Test ManagementState transitions, namespace immutability validation, DSC to MLFlow CR synchronization, upgrade scenarios

#### Post-MVP - Nice to Have

**REQ-TEST-004**: Chaos/Resilience Tests - Test recovery from deletion scenarios

**REQ-TEST-005**: Performance Tests - Test reconciliation performance and operator resource usage

### 6. Documentation Requirements

#### MVP - Must Have

**REQ-DOC-001**: API Documentation - GoDoc comments for all public types and methods with clear field descriptions and examples

**REQ-DOC-002**: Component README - Create `/docs/mlflow-component.md` with overview, architecture, configuration options, ManagementState behavior, troubleshooting

**REQ-DOC-003**: Operator Documentation Updates - Update main operator README with MLFlow component and version compatibility

**REQ-DOC-004**: CRD Reference Documentation - Generate CRD reference documentation with DSC configuration examples

#### Post-MVP - Nice to Have

**REQ-DOC-005**: User Guides - Getting started guide, admin guide, integration guide

**REQ-DOC-006**: Runbooks - Troubleshooting guide, upgrade guide, disaster recovery guide

### 7. Operational Requirements

#### MVP - Must Have

**REQ-OPS-001**: Resource Ownership - All deployed resources must have OwnerReferences to MLFlow CR for cascading deletion

**REQ-OPS-002**: Health Monitoring - Implement readiness and liveness probes; report deployment health via DeploymentsAvailable condition

**REQ-OPS-003**: Error Handling - Surface deployment errors in component status with actionable error messages and logging

**REQ-OPS-004**: Upgrades - Support in-place upgrades, preserve data during upgrades, validate compatibility before upgrading

#### Post-MVP - Nice to Have

**REQ-OPS-005**: Observability - Expose Prometheus metrics, provide Grafana dashboards, integrate with cluster logging

**REQ-OPS-006**: Backup/Restore - Document backup procedures and support disaster recovery scenarios

**Done - Acceptance Criteria:**

### Installation Scenarios

**AC-INST-001**: Deploy MLFlow via DSC
* **As a** cluster administrator
* **I can** set `spec.components.mlflow.managementState: Managed` in the DataScienceCluster CR
* **So that** MLFlow is automatically deployed and managed by the operator
* **Acceptance**: MLFlow CR created, namespace created, deployment running 1/1 pods, service accessible, DSC and CR status show Ready: True

**AC-INST-002**: Deploy MLFlow with Custom Namespace
* **As a** cluster administrator
* **I can** set `spec.components.mlflow.trackingNamespace: my-mlflow` in DSC
* **So that** MLFlow is deployed to my preferred namespace
* **Acceptance**: Custom namespace created, deployment in custom namespace, status reflects custom namespace

**AC-INST-003**: Deploy MLFlow via Direct CR Creation
* **As a** platform engineer
* **I can** create a MLFlow CR named `default-mlflow` directly
* **So that** I can manage MLFlow independently of DSC
* **Acceptance**: CR accepted, namespace created, deployment successful, status updated

**AC-INST-004**: Validation Prevents Invalid Configuration
* **As a** cluster administrator
* **I cannot** create a MLFlow CR with a name other than `default-mlflow`
* **So that** the singleton pattern is enforced
* **Acceptance**: Creation fails with clear validation error message

### Configuration Scenarios

**AC-CONF-001**: Namespace Immutability When Managed
* **As a** cluster administrator
* **I cannot** change `trackingNamespace` after MLFlow is deployed in Managed state
* **So that** data is not accidentally orphaned or lost
* **Acceptance**: Update rejected with validation error explaining immutability constraint

**AC-CONF-002**: Allow Namespace Change When Removed
* **As a** cluster administrator
* **I can** change `trackingNamespace` when MLFlow is in Removed state
* **So that** I can reconfigure before re-enabling
* **Acceptance**: Update accepted when ManagementState is Removed

**AC-CONF-003**: Status Reflects Actual State
* **As a** cluster administrator
* **I can** view MLFlow status in both MLFlow CR and DSC
* **So that** I understand the current deployment state
* **Acceptance**: Status visible via kubectl with Ready status, URL, conditions with timestamps, trackingNamespace field

### Lifecycle Scenarios

**AC-LIFE-001**: Enable Previously Removed Component
* **As a** cluster administrator
* **I can** change `managementState` from Removed to Managed
* **So that** MLFlow is deployed again
* **Acceptance**: Resources created, deployment ready, status transitions to Ready, data preserved

**AC-LIFE-002**: Remove Deployed Component
* **As a** cluster administrator
* **I can** change `managementState` from Managed to Removed
* **So that** MLFlow resources are cleaned up
* **Acceptance**: Deployment/service/ConfigMaps/Secrets deleted, namespace NOT deleted, status shows ManagementState: Removed

**AC-LIFE-003**: Update Resource Requests/Limits
* **As a** cluster administrator
* **I can** modify MLFlow deployment resources via DSC or CR
* **So that** I can right-size the deployment
* **Acceptance**: Deployment updated with new values, rolling update performed, no downtime, status remains Ready

**AC-LIFE-004**: Recover from Deployment Deletion
* **As a** cluster administrator or support engineer
* **I expect** the operator to automatically recreate deleted MLFlow deployments
* **So that** the system is self-healing
* **Acceptance**: Deployment recreated within 2 minutes, matches desired spec, status reflects recovery

**AC-LIFE-005**: Upgrade MLFlow Version
* **As a** cluster administrator
* **I can** update the operator to a new version with newer MLFlow
* **So that** I get security fixes and new features
* **Acceptance**: Operator upgrade triggers reconciliation, deployment updated with rolling update, data preserved, status reflects new release

### Failure Scenarios

**AC-FAIL-001**: Handle Deployment Failure - Status conditions show clear error messages when deployment fails

**AC-FAIL-002**: Handle Namespace Conflict - Status reflects namespace creation failure with actionable error message

**AC-FAIL-003**: Handle Resource Constraints - Status indicates scheduling failure with reason and allows automatic recovery

**AC-FAIL-004**: Validation Error on Invalid Namespace Name - CR update rejected with clear validation error and RFC 1123 requirements

### Observability Scenarios

**AC-OBS-001**: Monitor Component Health via CLI - `kubectl get mlflow` shows Ready/Reason/URL columns

**AC-OBS-002**: View Detailed Status - `kubectl describe mlflow` shows detailed conditions, status, events

**AC-OBS-003**: Track Release Versions - Status includes releases array with accurate version information

### Integration Scenarios

**AC-INTG-001**: DSC Synchronization - Changes to DSC propagate to MLFlow CR; status flows back to DSC

**AC-INTG-002**: Resource Ownership Chain - All resources have correct OwnerReferences; namespace does NOT have OwnerReference

**AC-INTG-003**: DSCInitialization Dependency - Changes to DSCInitialization trigger reconciliation; MLFlow respects platform settings

**Use Cases - i.e. User Experience & Workflow:**

### Use Case 1: Deploy MLFlow for the First Time

**Actor**: Cluster Administrator

**Preconditions**: RHOAI operator installed, DataScienceCluster CR exists

**Main Success Scenario**:

1. Administrator enables MLFlow via DSC:
```yaml
kubectl edit datasciencecluster default
# Update:
spec:
  components:
    mlflow:
      managementState: Managed
```

2. Administrator monitors deployment:
```bash
kubectl get dsc default -w
kubectl get mlflow default-mlflow -w
kubectl describe mlflow default-mlflow
```

Expected output:
```
NAME              READY   REASON      URL
default-mlflow    True    Available   http://mlflow-tracking.odh-mlflow.svc.cluster.local:5000
```

3. Administrator verifies deployment:
```bash
kubectl get namespace odh-mlflow
kubectl get pods -n odh-mlflow
kubectl get svc -n odh-mlflow
```

Expected: Namespace exists, deployment 1/1 ready, service available

4. Administrator validates MLFlow is accessible:
```bash
kubectl port-forward -n odh-mlflow svc/mlflow-tracking 5000:5000
curl http://localhost:5000/health
```

Expected: MLFlow UI loads or health endpoint returns 200 OK

**Postconditions**: MLFlow deployed and healthy, status Ready: True, tracking server accessible, teams can start logging experiments

**Alternative Flow 1A**: Custom Namespace - Specify custom `trackingNamespace`, resources created in custom namespace

**Alternative Flow 1B**: Deployment Fails - Status shows error condition with clear message; administrator resolves (add capacity, reduce resources); operator automatically recovers

### Use Case 2: Upgrade Existing MLFlow Instance

**Actor**: Cluster Administrator

**Preconditions**: MLFlow deployed and running, data science teams actively using it

**Main Success Scenario**:

1. Administrator reviews upgrade notes and backs up critical data

2. Administrator upgrades RHOAI operator (via OLM or manual deployment update)

3. Operator automatically reconciles MLFlow with new version

4. Administrator monitors upgrade progress:
```bash
kubectl get mlflow default-mlflow -w
kubectl rollout status -n odh-mlflow deployment/mlflow-tracking-server
```

Expected: Status shows Progressing then Available

5. Administrator verifies upgraded version:
```bash
kubectl get mlflow default-mlflow -o jsonpath='{.status.releases}'
kubectl get pods -n odh-mlflow
```

6. Administrator validates functionality and confirms with teams

**Postconditions**: MLFlow running new version, all data accessible, status reflects new release, no downtime

**Alternative Flow 2A**: Upgrade Fails - Status shows ImagePullBackOff; administrator resolves registry/image issues; automatic reconciliation completes

**Alternative Flow 2B**: Rollback Required - Revert operator to previous version; operator reconciles MLFlow back to previous version

### Use Case 3: Remove MLFlow Component

**Actor**: Cluster Administrator

**Preconditions**: MLFlow deployed, teams notified, data backed up

**Main Success Scenario**:

1. Administrator disables MLFlow via DSC:
```yaml
spec:
  components:
    mlflow:
      managementState: Removed
```

2. Operator reconciles removal (deletes deployment, service, ConfigMaps, Secrets)

3. Administrator monitors removal:
```bash
kubectl get mlflow default-mlflow -w
kubectl get all -n odh-mlflow
```

Expected: Status shows Ready: False, Reason: Removed

4. Administrator verifies cleanup:
```bash
kubectl get deployment -n odh-mlflow mlflow-tracking-server  # NotFound
kubectl get namespace odh-mlflow  # Still exists
```

**Postconditions**: MLFlow resources removed, namespace preserved with user data, status shows Removed

**Alternative Flow 3A**: Complete Cleanup - Manually delete namespace after removal for complete data deletion

### Use Case 4: Troubleshoot Failed Deployment

**Actor**: SRE or Cluster Administrator

**Preconditions**: MLFlow enabled but deployment not ready

**Main Success Scenario**:

1. SRE checks high-level status:
```bash
kubectl get dsc default
kubectl get mlflow default-mlflow
```

Sees: Ready: False, Reason: DeploymentsUnavailable

2. SRE examines detailed status:
```bash
kubectl describe mlflow default-mlflow
```

Sees conditions with error messages

3. SRE drills down to deployment:
```bash
kubectl get deployment -n odh-mlflow mlflow-tracking-server
kubectl get pods -n odh-mlflow
kubectl describe pods -n odh-mlflow
```

Identifies root cause (e.g., ImagePullBackOff, InsufficientResources, configuration error)

4. SRE resolves issue (fix image tag, add capacity, correct configuration)

5. Operator automatically recovers

6. SRE verifies resolution:
```bash
kubectl get mlflow default-mlflow
kubectl exec -n odh-mlflow deployment/mlflow-tracking-server -- curl http://localhost:5000/health
```

Expected: Status Ready: True

**Postconditions**: MLFlow running and healthy, root cause documented, permanent fix tracked

**Alternative Flows**: Configuration errors, resource constraints, operator-level issues requiring log analysis

**Documentation Considerations:**

### User Documentation (Data Scientists)

* **Getting Started Guide**: How to enable MLFlow, verify it's running, connect from notebook
* **MLFlow Client Configuration**: Python/R client setup for tracking server connection
* **Experiment Tracking Tutorials**: Step-by-step examples for logging parameters, metrics, models
* **Artifact Management**: How to log and retrieve model artifacts, datasets, plots
* **Integration Examples**: Using MLFlow from JupyterLab, logging pipeline runs, deploying to KServe
* **Troubleshooting Guide**: Common issues (connection failures, authentication errors, storage problems)

### Platform Administrator Documentation

* **Installation Guide**: Enabling MLFlow component in DataScienceCluster CR
* **Configuration Reference**: Complete API documentation for MLFlow CR fields
* **Storage Configuration**: Setting up external databases and S3-compatible storage, credential management
* **Multi-Tenancy Setup**: Provisioning isolated MLFlow instances for different teams
* **Resource Management**: Sizing guidance, quota configuration, resource limits
* **Security Configuration**: Authentication/authorization setup, network policies, encryption
* **Upgrade Guide**: Upgrading MLFlow including database migration steps
* **Backup and Disaster Recovery**: Procedures for backing up MLFlow data
* **Monitoring and Alerting**: Prometheus metrics, Grafana dashboards, alert runbooks

### Architecture & Design Documentation

* **Component Architecture**: High-level design of MLFlow integration in RHOAI
* **Data Flow Diagrams**: How experiments, metrics, artifacts flow through system
* **Integration Architecture**: How MLFlow connects with other RHOAI components
* **Multi-Tenancy Model**: Detailed explanation of isolation boundaries
* **Storage Backend Architecture**: Database and object storage design decisions
* **API Reference**: Complete OpenAPI spec for MLFlow CR

### Troubleshooting Documentation

* **Diagnostic Runbooks**: Tracking server not starting, database connection failures, storage backend errors, client connection issues, performance degradation, upgrade failures
* **Status Condition Reference**: Complete table of conditions with meanings, reasons, remediation steps

### Migration & Adoption Documentation

* **Migration Guides**: From self-managed MLFlow, between storage backends, data export/import procedures
* **Adoption & Best Practices**: Experiment organization, model lifecycle management, team collaboration, performance optimization, cost management

### Examples & Tutorials

* **Configuration Examples**: Minimal deployment, production-ready with external DB/S3, multi-tenant, HA
* **Code Examples**: Python client configuration, integration with PyTorch/TensorFlow, logging custom metrics, model deployment to KServe

**Questions to answer:**

### Storage Architecture & Data Persistence

**Q1**: What is the storage strategy for MLFlow metadata and artifact persistence?
* Should we follow Data Science Pipelines pattern with embedded (SQLite/ephemeral) and external (PostgreSQL/MySQL) support?
* Do we provide object storage configuration (S3/MinIO) as part of component CR or expect separate configuration?
* For multi-instance scenarios: dedicated database/storage per instance or shared with schema isolation?
* Are we providing production-grade storage out of the box or a "bring your own database/storage" model?

**Q2**: How do we handle artifact storage backend configuration and credentials?
* Auto-provision MinIO per deployment (heavy but isolated) or provide shared MinIO/S3 with bucket-level isolation?
* How are storage credentials managed - operator-managed secrets, user-provided secrets, or external secret management integration?
* Should operator validate storage connectivity before marking MLFlow as Ready?
* What's the upgrade story when artifact locations change - do we support artifact migration?

**Q3**: What are the disaster recovery and backup requirements for MLFlow data?
* Does component status need to surface backup/restore capabilities?
* Are we documenting operational procedures for database and artifact backup?
* How do we handle StatefulSet-based deployments if HA is required?

### Multi-Tenancy & Isolation Model

**Q4**: What is the multi-tenancy model - namespace-per-instance, single namespace with multiple deployments, or cluster-wide shared service?
* Looking at ModelRegistry (namespace for all registries) vs Workbenches (single namespace): which pattern fits MLFlow?
* Multiple MLFlow CRs per team with separate tracking servers, or one CR managing multiple instances?
* How does this align with RHOAI's project/namespace isolation model used by Dashboard?
* Should we support "shared MLFlow service" mode with auth-based isolation?

**Q5**: How do we enforce access control and authentication between MLFlow instances?
* Does each tracking server need OAuth proxy integration (like ModelRegistry) or rely on network policies and RBAC?
* How do users authenticate - service accounts, OpenShift users, external identity providers?
* For multi-instance scenarios, how do we prevent cross-instance data access?
* Do we need to integrate with RHOAI's existing Auth service CR?

**Q6**: What are the resource quota and limit strategies for MLFlow deployments?
* Should MLFlow CR expose resource requests/limits configuration or use opinionated defaults?
* How do we prevent resource exhaustion in multi-tenant scenarios - ResourceQuotas per namespace, LimitRanges?
* What's the guidance for sizing MLFlow based on expected usage (users, experiments, artifacts)?

### Scalability & Performance

**Q7**: What are the performance and scale requirements for MLFlow tracking servers?
* Target: 10 users, 100 users, 1000+ users per instance?
* Do we need horizontal scaling (multiple tracking server replicas with shared database)?
* Performance impact of large numbers of experiments (10K, 100K, 1M+) on tracking server and database?
* Should we expose database connection pooling and caching configuration?

**Q8**: How do we handle large artifact storage scenarios?
* Expected artifact size - MBs, GBs, TBs?
* Do we need to document storage growth projections and capacity planning?
* Should operator monitor artifact storage usage and surface it in status?
* Are there lifecycle management considerations (artifact retention policies, archival)?

### Integration with RHOAI Ecosystem

**Q9**: How does MLFlow integrate with existing RHOAI components (Workbenches, Data Science Pipelines, Model Registry, KServe)?
* Should Workbenches have MLFlow client pre-configured with tracking URI?
* Can Data Science Pipelines automatically log runs to MLFlow?
* Does MLFlow's model registry functionality overlap/conflict with RHOAI's ModelRegistry, and how do we handle that?
* Can models logged in MLFlow be directly deployed to KServe?
* Do we need custom integration layer or are these user-configured integrations?

**Q10**: What dependencies does MLFlow require, and how do we manage them?
* Does MLFlow require any CRDs (like KServe requires Knative/Istio)?
* Are there version compatibility matrices (Python client vs tracking server)?
* How do we handle upgrades when tracking server version changes - is it compatible with existing data?
* Should we vendor MLFlow or pull from upstream containers?

### High Availability & Reliability

**Q11**: Do we need to support high availability for MLFlow tracking servers?
* Is this "nice to have" or "must have" for production deployments?
* If HA required: active-active or active-passive? StatefulSet or Deployment?
* How do we handle database HA - user's responsibility or integrate with operators like Crunchy PostgreSQL?
* What are RTO/RPO requirements for MLFlow?

**Q12**: How do we handle failure scenarios and self-healing?
* If tracking server crashes, do experiments fail or queue?
* What health checks and readiness probes should we implement?
* Should operator auto-recover from database connection failures?
* How do we surface error conditions in status (database unavailable, storage full)?

### Security & Compliance

**Q13**: What are the security requirements for MLFlow deployments?
* Do we need Pod Security Standards enforcement (restricted profile)?
* Should artifact storage be encrypted at rest? How do we handle encryption keys?
* Are there network policy requirements to isolate MLFlow traffic?
* Do we need audit logging for experiment tracking and model logging?
* How do we handle sensitive data in experiment parameters/metrics (secrets, PII)?

**Q14**: How do we handle secret management for database credentials and storage access?
* Operator-generated secrets vs user-provided secrets?
* Secret rotation strategy?
* Integration with external secret stores?
* How are secrets propagated to user workbenches for MLFlow client configuration?

### Upgrades & Version Management

**Q15**: What is the upgrade strategy for MLFlow component and tracking server versions?
* Can we upgrade tracking server without downtime?
* How do we handle database schema migrations during upgrades?
* Is there a rollback strategy if upgrade fails?
* Do we support rolling upgrades for multi-replica deployments?
* What's the version skew policy between MLFlow server and Python client?

**Q16**: How do we handle data migration when storage backends change?
* If user switches from SQLite to PostgreSQL or between S3 buckets?
* Do we provide migration tools/documentation or is this unsupported?
* How do we minimize data loss risk during migrations?

### Observability & Operations

**Q17**: What metrics and alerts do we need for MLFlow operational health?
* Critical metrics: tracking server availability/uptime, database connection health, storage backend health, API request latency/error rates, artifact upload/download success rates
* What alerts indicate user-facing vs operator-facing issues?
* Should we expose experiment/run metrics (active experiments, run creation rate)?

**Q18**: How do users troubleshoot MLFlow issues?
* What logs should operator and tracking server expose?
* How do we surface configuration errors (wrong credentials, unreachable database)?
* Do we need status condition for common issues (StorageUnavailable, DatabaseMigrationFailed)?
* Should we provide kubectl plugin or CLI tool for MLFlow diagnostics?

### Customization & Configuration

**Q19**: What level of customization should we expose in MLFlow CR vs keeping opinionated?
* Tracking server configuration (artifact locations, authentication backends)?
* Image versions for tracking server?
* Database connection parameters (pool size, timeouts)?
* Storage backend configuration (region, endpoint, bucket naming)?
* Looking at TrustyAI's approach with LMEval configuration: should we have `mlflow.server` nested configuration?

**Q20**: How do we handle namespace configuration and immutability constraints?
* Following ModelRegistry pattern: `trackingNamespace` is immutable when Managed - is this the right constraint?
* Should we allow changing namespace if no MLFlow instances are running?
* What happens if users try to deploy multiple MLFlow CRs with different namespace configurations?

### API Design & User Experience

**Q21**: What should the MLFlow CR API surface look like?
* Minimal: just `trackingNamespace` and `ManagementState`?
* Moderate: add database and storage backend configuration?
* Comprehensive: expose full MLFlow server configuration?
* Do we need `spec.replicas` field for HA or is that an advanced/future feature?

**Q22**: How do users discover and connect to their MLFlow tracking server?
* Do we expose Route/Ingress automatically?
* How is tracking server URL communicated (status field, ConfigMap, Dashboard integration)?
* Should workbenches get environment variables auto-configured with `MLFLOW_TRACKING_URI`?
* Do we support both UI access (web dashboard) and API access (Python client)?

**Background & Strategic Fit:**

**Why MLFlow Matters to RHOAI Strategy:**

According to recent market data, 68% of enterprises cite "experiment tracking and reproducibility" as a top-3 MLOps challenge. MLFlow is the de facto standard in this space, with:
* 5M+ downloads per month on PyPI
* Adoption by 70% of Fortune 500 companies with active ML initiatives
* Native integrations with popular ML frameworks (TensorFlow, PyTorch, scikit-learn, XGBoost)

**Strategic Fit with RHOAI:**

1. **Complete the MLOps Lifecycle Story** - Today, RHOAI excels at model development (Workbenches) and model serving (KServe/ModelMesh), but has a gap in experiment management. Our customers tell us: "We love RHOAI for production ML, but our data scientists still run MLFlow on their laptops or separate clusters because there's no integrated tracking." Adding MLFlow closes this gap and positions RHOAI as a complete MLOps platform.

2. **Differentiation from Cloud Provider ML Platforms** - AWS SageMaker, Azure ML, and GCP Vertex AI all include experiment tracking as core features. Without MLFlow (or equivalent), RHOAI appears incomplete compared to these platforms, especially in hybrid cloud scenarios where customers want platform portability. Open-source MLFlow on OpenShift becomes a competitive advantage: "Run the same ML workflows on-prem and in any cloud without vendor lock-in."

3. **Synergy with Existing Components**:
   * **Workbenches**: Data scientists track experiments from notebooks; MLFlow becomes the "memory" of their work
   * **ModelRegistry**: MLFlow stores experiment artifacts/metrics; ModelRegistry handles production model governance (clear handoff point)
   * **KServe**: MLFlow produces model artifacts; KServe deploys them to production
   * **Dashboard**: MLFlow UI integrated into Dashboard provides unified UX without context switching
   * **Data Science Pipelines**: Pipeline runs can log metrics/artifacts to MLFlow for centralized tracking

4. **Enterprise Requirements Alignment**:
   * **Compliance**: Enterprises need audit trails of model experiments for regulatory purposes (GDPR, CCPA, AI Act). MLFlow provides this.
   * **Reproducibility**: Ability to recreate any model from historical experiments is critical for regulated industries (finance, healthcare, pharma).
   * **Collaboration**: Teams need to share experiment results across geographies. MLFlow server provides centralized storage.

**Customer Evidence** (anonymized):
* "We deployed 15 separate MLFlow instances manually across our RHOAI clusters. It's a nightmare to keep them updated." - Financial Services, F100
* "Our data scientists love RHOAI notebooks but immediately ask, 'Where do I log my experiments?' We point them to external MLFlow servers, and they wonder why it's not integrated." - Healthcare, F500
* "We can't adopt RHOAI fully until we have experiment tracking. We're currently evaluating competitors who bundle this." - Retail, F200

**Market Risk if We Don't Deliver**:
* **Competitor Advantage**: Kubeflow Pipelines includes experiment tracking; customers comparing RHOAI to Kubeflow see this as a gap
* **Shadow IT**: Data scientists will deploy MLFlow themselves in unsupported configurations, creating security/compliance risks
* **Adoption Friction**: Customers delay/reduce RHOAI adoption because they need to integrate external experiment tracking tools

**Bottom Line**: MLFlow integration is a strategic imperative to position RHOAI as a complete, enterprise-grade MLOps platform that can compete with cloud-native offerings and prevent customer churn to alternative solutions.

**Customer Considerations:**

**Critical Customer-Specific Needs:**

**1. Multi-Tenancy and RBAC**
* **Need**: Enterprises have multiple ML teams needing isolated MLFlow instances with namespace-level access control
* **Requirement**: Support multiple MLFlow instances per cluster with namespace-scoped RBAC; Team A should not see Team B's experiments
* **Implementation**: Align with RHOAI's existing multi-tenancy model (namespace-per-team); MLFlow instances inherit namespace RBAC policies
* **Customer Quote**: "We have 8 data science teams. We can't give them all access to the same MLFlow server - it's a compliance violation."

**2. Migration from Existing MLFlow Deployments**
* **Need**: Many customers already run MLFlow and need migration path without data loss
* **Requirement**: Provide documentation (not tooling) for migrating experiment data from external MLFlow to RHOAI-managed MLFlow; include supported sources: filesystem backends, MySQL/PostgreSQL databases, S3-compatible storage
* **Risk**: If migration is too complex, customers won't adopt operator-managed MLFlow
* **Customer Quote**: "We have 2 years of experiment history. If we can't migrate that, we're not switching."

**3. Integration with Existing Storage Infrastructure**
* **Need**: Enterprises have standardized on specific storage backends (NetApp for PVCs, MinIO for objects, enterprise PostgreSQL for metadata)
* **Requirement**: MLFlow must support customer-managed databases and artifact storage via configuration, not just operator-provisioned defaults
* **Trade-off**: Adds complexity but is non-negotiable for large enterprises with strict storage policies
* **Customer Quote**: "We can't let every application spin up its own database. MLFlow must use our centrally-managed PostgreSQL cluster."

**4. Compliance and Audit Requirements**
* **Need**: Regulated industries require audit trails of who accessed which experiments, when, and what changes were made
* **Requirement**: MLFlow deployment must integrate with OpenShift's audit logging; all API calls to MLFlow server should be traceable
* **Additional Need**: Support for immutable experiment logs (prevent deletion/modification of historical data after retention period)
* **Customer Quote**: "For FDA validation, we need to prove no one altered experiment results post-facto. Audit logs are mandatory."

**5. Air-Gapped and Disconnected Environments**
* **Need**: Defense, government, high-security enterprises run RHOAI in air-gapped environments with no internet access
* **Requirement**: MLFlow operator must work in disconnected OpenShift clusters; all container images must be mirrorable to internal registries; no external dependencies at runtime
* **Implementation**: Follow RHOAI's existing disconnected installation patterns
* **Customer Quote**: "We operate in a classified environment. Any component that phones home or requires internet access is a non-starter."

**6. Performance at Scale**
* **Need**: Large organizations have hundreds of data scientists running thousands of experiments per day
* **Requirement**: MLFlow operator must support horizontal scaling (multiple replicas behind load balancer) to handle high API request volumes
* **Metrics**: Support 100+ concurrent users, 10,000+ experiment runs per day, 1TB+ artifact storage per instance
* **Customer Quote**: "Our current MLFlow server crashes every week because it can't handle the load. We need something production-grade."

**7. Disaster Recovery and Backup**
* **Need**: Experiment data represents months/years of R&D investment; loss of this data is catastrophic
* **Requirement**: Integrate with OpenShift's backup solutions (OADP, Velero); MLFlow instances must be backup-able and restore-able with minimal operator intervention
* **Documentation**: Provide runbooks for backup/restore procedures including database and artifact storage
* **Customer Quote**: "We lost 6 months of experiments in a storage failure last year. Backups are now a hard requirement for any ML tooling."

**8. Integration with Enterprise Identity Providers**
* **Need**: Customers use corporate identity systems (Active Directory, LDAP, Okta, Azure AD) for authentication
* **Requirement**: MLFlow authentication must integrate with OpenShift OAuth which can federate to enterprise IdPs; no separate username/password management
* **UX Note**: Users should SSO into RHOAI Dashboard and access MLFlow without re-authenticating
* **Customer Quote**: "We don't want another password to manage. If it doesn't use our corporate SSO, it's a security risk."

**Prioritization for MVP:**
* **P0 (Must-Have)**: Multi-tenancy/RBAC, disconnected environment support, OpenShift OAuth integration
* **P1 (Should-Have)**: Custom storage backends, horizontal scaling, audit logging integration
* **P2 (Nice-to-Have)**: Migration tooling (documentation-only for MVP), advanced backup automation
