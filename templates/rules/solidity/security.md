# Solidity Security

## Reentrancy Protection

- Apply `nonReentrant` modifier (OpenZeppelin ReentrancyGuard) on all state-changing external functions
- Follow CEI pattern as the primary defense; reentrancy guard as secondary
- Be aware of cross-function and cross-contract reentrancy
- Read-only reentrancy: guard against reentrancy in view functions that read inconsistent state

## Access Control

- Use OpenZeppelin's `AccessControl` or `Ownable2Step` (not plain `Ownable`)
- `Ownable2Step` requires the new owner to accept, preventing accidental transfers
- Use role-based access control for contracts with multiple admin functions
- Implement timelock for sensitive admin operations (parameter changes, upgrades)

## Authentication

- **Never use `tx.origin`** for authentication; always use `msg.sender`
- `tx.origin` is vulnerable to phishing attacks via malicious intermediary contracts
- For meta-transactions, validate signatures with EIP-712 typed data

## Payment Patterns

- **Pull over push**: let recipients withdraw, don't send to them
- Use `call` with value instead of `transfer` or `send` (gas limit issues)
- Always check return value of `call`: `(bool success, ) = addr.call{value: amount}("")`
- Handle failed transfers gracefully; don't let one failure block others

## Integer Safety

- Solidity 0.8+ has built-in overflow/underflow checks
- Use `unchecked` blocks only with explicit comments proving safety
- Be aware of precision loss in division; multiply before dividing
- Use fixed-point math libraries for financial calculations

## Upgrade Safety

- Use OpenZeppelin's UUPS or Transparent Proxy patterns
- Never use `selfdestruct` or `delegatecall` to arbitrary addresses
- Maintain storage layout compatibility between upgrades (no reordering, no removal)
- Use storage gaps in base contracts: `uint256[50] private __gap`
- Test upgrade paths with OpenZeppelin Upgrades plugin

## Input Validation

- Validate all external inputs: zero addresses, zero amounts, array lengths
- Check array length bounds to prevent out-of-gas on iteration
- Validate that addresses are contracts when expected (`address.code.length > 0`)
- Use `SafeERC20` for token interactions (handles non-standard return values)

## Front-Running Protection

- Use commit-reveal schemes for sensitive operations (auctions, votes)
- Consider MEV impact on AMM and trading functions
- Use deadline parameters for time-sensitive transactions
- Implement slippage protection for swap operations

## External Calls

- Treat all external calls as untrusted, even to "known" contracts
- Limit gas forwarded to external calls when appropriate
- Handle revert data from failed calls for better error reporting
- Never assume external contract behavior won't change (behind proxies)

## Audit Checklist

- [ ] All state changes follow CEI pattern
- [ ] Reentrancy guards on state-changing functions
- [ ] Access control on admin functions
- [ ] Events emitted for all state changes
- [ ] No `tx.origin` usage
- [ ] Pull payment pattern for ETH transfers
- [ ] Input validation on all external functions
- [ ] Storage layout compatible with upgrades
