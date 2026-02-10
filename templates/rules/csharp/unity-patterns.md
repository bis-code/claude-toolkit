# Unity / C# Game Patterns

## Component Architecture

- Prefer composition over deep MonoBehaviour inheritance hierarchies
- One responsibility per component; combine via GameObjects
- Use `[RequireComponent]` to declare dependencies between components
- Avoid god-objects: split large MonoBehaviours into focused components

## ScriptableObjects

- Use ScriptableObjects for game configuration and shared data
- Store balancing values, item definitions, and ability configs as assets
- Use ScriptableObject events for decoupled communication between systems
- Never store runtime state in ScriptableObjects (they persist across play sessions)

## Object Pooling

- Pool frequently instantiated objects (projectiles, particles, enemies)
- Use `ObjectPool<T>` (Unity 2021+) or implement a custom pool
- Pre-warm pools at scene load, not during gameplay
- Reset object state on return to pool, not on retrieval

## Performance

- Cache component references in `Awake()` or `Start()`; never call `GetComponent` in `Update()`
- **Never use `FindObjectOfType`** or `GameObject.Find` in runtime code
- Use `CompareTag()` instead of `tag == "string"` (avoids allocation)
- Minimize allocations in hot loops; avoid LINQ in `Update()`
- Use `NonAlloc` variants: `Physics.RaycastNonAlloc`, `Physics.OverlapSphereNonAlloc`

## Coroutines vs Async

- Use coroutines for simple timed sequences and frame-by-frame work
- Cache `WaitForSeconds` instances; don't create new ones every call
- Use async/await with UniTask for complex async flows
- Never mix `async void` with Unity lifecycle methods

## Prefabs

- Use prefab variants for shared base + specialized behavior
- Keep prefab hierarchies shallow (max 2-3 levels of nesting)
- Use addressables for prefabs loaded at runtime (not `Resources.Load`)
- Avoid serialized references to scene objects from prefabs

## Events and Communication

- Use C# events or UnityEvents for component-to-component communication
- Use a lightweight event bus or ScriptableObject events for cross-system communication
- Avoid direct references between unrelated systems
- Unsubscribe from events in `OnDisable()` or `OnDestroy()`

## Editor and Tooling

- Use `[SerializeField]` for private fields that need Inspector exposure
- Use `[Header]` and `[Tooltip]` attributes for designer-facing components
- Write custom editors for complex components used by non-programmers
- Use `#if UNITY_EDITOR` for editor-only code

## Scene Management

- Use additive scene loading for modular level design
- Keep a persistent "Bootstrap" scene for managers and singletons
- Limit singletons; prefer dependency injection or service locators
- Use ScriptableObject-based scene references over build index numbers

## Project Structure

```
Assets/
  _Project/          # All project-specific assets
    Scripts/
      Core/          # Managers, utilities, base classes
      Gameplay/      # Game-specific logic
      UI/            # UI controllers and views
    Prefabs/
    ScriptableObjects/
    Art/
  Plugins/           # Third-party assets
```
