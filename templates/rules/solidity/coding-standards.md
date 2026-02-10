# Solidity Coding Standards

## CEI Pattern (Checks-Effects-Interactions)

Every state-changing function must follow this order:
1. **Checks**: Validate inputs and conditions (require, custom errors)
2. **Effects**: Update contract state
3. **Interactions**: External calls (transfers, cross-contract calls)

Never make external calls before updating state.

## Custom Errors

- Use custom errors over `require` strings (significantly cheaper gas)
- Name errors descriptively: `error InsufficientBalance(uint256 requested, uint256 available)`
- Define errors at the contract level, not inside functions
- Group common errors in a shared `Errors.sol` file

## NatSpec Documentation

- Document every public and external function with NatSpec
- Use `@notice` for user-facing explanation, `@dev` for developer context
- Document all parameters with `@param` and return values with `@return`
- Document events and custom errors

```solidity
/// @notice Transfers tokens to a recipient
/// @param to The recipient address
/// @param amount The number of tokens to transfer
/// @return success Whether the transfer succeeded
function transfer(address to, uint256 amount) external returns (bool success);
```

## State Variables

- Use `immutable` for values set once in the constructor
- Use `constant` for compile-time known values
- Order storage variables to pack efficiently (32-byte slots)
- Use `private` by default; expose via getter functions when needed

## Events

- Emit events for every state change (transfers, approvals, config updates)
- Index parameters that will be filtered on (max 3 indexed per event)
- Include both old and new values for state transitions
- Event names should be past-tense verbs: `Transferred`, `Approved`, `RoleGranted`

## Modifiers

- Keep modifiers simple; complex logic belongs in functions
- Common modifiers: `onlyOwner`, `whenNotPaused`, `nonReentrant`
- Apply modifiers in a consistent order across functions
- Avoid deep modifier stacking (max 2-3 per function)

## Gas Optimization

- Use `calldata` over `memory` for external function parameters
- Cache storage reads in local variables within functions
- Use `unchecked` blocks for math that cannot overflow (with comment explaining why)
- Prefer mappings over arrays for lookups; arrays for iteration
- Avoid dynamic arrays in storage when possible

## Contract Organization

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

// 1. Imports
// 2. Errors
// 3. Interfaces
// 4. Libraries
// 5. Contract
//    a. Type declarations (enums, structs)
//    b. State variables
//    c. Events
//    d. Modifiers
//    e. Constructor
//    f. External functions
//    g. Public functions
//    h. Internal functions
//    i. Private functions
```

## Versioning

- Pin pragma to a specific minor version: `pragma solidity ^0.8.20;`
- Use the latest stable compiler version for new projects
- Test with the exact compiler version used for deployment
