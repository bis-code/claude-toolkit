# Solidity Testing Standards

## Framework

- Use Foundry as the primary testing framework
- Write tests in Solidity (not JavaScript) for type safety and speed
- Use `forge test` for running tests, `forge coverage` for coverage
- Use Hardhat only when JavaScript tooling is specifically required

## Test Structure

```solidity
contract TokenTest is Test {
    Token token;
    address alice = makeAddr("alice");
    address bob = makeAddr("bob");

    function setUp() public {
        token = new Token();
        deal(address(token), alice, 1000e18);
    }

    function test_transfer_success() public {
        vm.prank(alice);
        token.transfer(bob, 100e18);
        assertEq(token.balanceOf(bob), 100e18);
    }

    function test_transfer_insufficientBalance_reverts() public {
        vm.prank(alice);
        vm.expectRevert(Token.InsufficientBalance.selector);
        token.transfer(bob, 2000e18);
    }
}
```

## Naming Conventions

- Test contracts: `<Contract>Test`
- Success tests: `test_<function>_<scenario>`
- Revert tests: `test_<function>_<scenario>_reverts`
- Use descriptive scenario names, not just `test_transfer`

## Fuzz Testing

- Use fuzz tests for functions accepting numeric or bytes inputs
- Bound inputs to realistic ranges with `vm.assume()` or `bound()`
- Run with sufficient iterations: `forge test --fuzz-runs 10000`
- Fuzz tests should cover invariants, not just happy paths

```solidity
function testFuzz_transfer(uint256 amount) public {
    amount = bound(amount, 1, token.balanceOf(alice));
    vm.prank(alice);
    token.transfer(bob, amount);
    assertEq(token.balanceOf(bob), amount);
}
```

## Invariant Testing

- Define invariants that must hold across all possible function call sequences
- Use `targetContract`, `targetSelector` to scope the test
- Test global invariants: total supply consistency, balance sum equality
- Run with `forge test --mt invariant`

```solidity
function invariant_totalSupplyMatchesBalances() public {
    assertEq(token.totalSupply(), token.balanceOf(alice) + token.balanceOf(bob));
}
```

## Fork Testing

- Use `vm.createFork()` for testing against mainnet state
- Pin to a specific block number for deterministic tests
- Test integrations with deployed protocols (Uniswap, Aave, etc.)
- Use `--fork-url` with a cached RPC for speed

## Gas Snapshots

- Use `forge snapshot` to track gas usage over time
- Commit `.gas-snapshot` file to the repository
- Review gas changes in PRs; investigate unexpected increases
- Use `--diff` to compare against the baseline snapshot

## Cheatcodes (Essential)

| Cheatcode | Purpose |
|-----------|---------|
| `vm.prank(addr)` | Set msg.sender for next call |
| `vm.deal(addr, amount)` | Set ETH balance |
| `deal(token, addr, amount)` | Set ERC20 balance |
| `vm.warp(timestamp)` | Set block.timestamp |
| `vm.roll(blockNum)` | Set block.number |
| `vm.expectRevert()` | Assert next call reverts |
| `vm.expectEmit()` | Assert event emission |
| `makeAddr(name)` | Create labeled address |

## Test Organization

```
test/
  unit/
    Token.t.sol
    Staking.t.sol
  integration/
    TokenStaking.t.sol
  invariant/
    TokenInvariant.t.sol
  fork/
    MainnetIntegration.t.sol
  utils/
    TestHelper.sol
```

## Coverage

- Target 90%+ line coverage for core contracts
- Use `forge coverage --report lcov` for detailed reports
- Focus coverage on state-changing and access-controlled functions
- Uncovered code in production contracts must have justification
