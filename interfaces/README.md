# TxWatch Interfaces

This package defines the pluggable interfaces for TxWatch fraud detection.

## Core Interfaces

### TransactionRepository
Provides access to transaction data. Implement this to connect TxWatch to your data storage.

**Methods:**
- `FetchHistorical`: Retrieve historical transactions for risk aggregation
- `Persist`: Store flagged transactions for reporting

**Thread Safety:** Implementations MUST be thread-safe.

### AIModel
Provides AI-based fraud scoring. Implement this to integrate your ML models.

**Methods:**
- `Score`: Compute risk score (0.0-1.0) for a transaction

**Requirements:**
- Scores MUST be between 0.0 and 1.0
- Implementations SHOULD be stateless
- MUST be thread-safe for concurrent scoring

### AlertHandler
Provides notification capabilities for flagged transactions.

**Methods:**
- `Alert`: Send notification for flagged transaction

**Requirements:**
- SHOULD be idempotent for retry scenarios
- MUST be thread-safe

### Rule
Defines a fraud detection rule for evaluating transactions.

**Methods:**
- `ID`: Unique identifier
- `Name`: Human-readable name
- `Description`: What the rule checks
- `Evaluate`: Assess transaction for fraud

**Requirements:**
- Rules are executed independently
- MUST be thread-safe
- Evaluation order should not matter

### AggregationStrategy
Defines how multiple risk signals are combined into a single score.

**Methods:**
- `Aggregate`: Combine risk signals into final score

**Requirements:**
- Output MUST be between 0.0 and 1.0
- SHOULD handle empty inputs gracefully

## Implementation Guidelines

1. **Thread Safety**: All interfaces must be safe for concurrent use
2. **Context Handling**: Respect context cancellation and timeouts
3. **Error Handling**: Return descriptive errors for debugging
4. **Idempotency**: Where applicable, handle duplicate calls gracefully
5. **No Global State**: Avoid mutable global state in implementations
