extends RefCounted

## Base class for e2e scenario tests. Override scenario_name() and run().
## Scenarios are await-based coroutines driven by an E2EContext.


func scenario_name() -> String:
	return "unnamed"


func run(_ctx: RefCounted) -> void:
	_ctx.fail("scenario did not override run()")
