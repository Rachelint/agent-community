// Package plugin manages assistant and worker plugin processes.
//
// Both chat assistants and workers are modeled symmetrically: an external
// process described by a JSON manifest under assistants/<name>/assistant.json,
// spawned with stdio piped back to the host. Communication uses JSON-RPC
// line-delimited messages. The host verifies startup, then steps back;
// plugins drive progress by sending notifications (worker.log,
// worker.complete, chat.delta, chat.done, ...).
//
// Phase 0 placeholder; implementation arrives in phase 3.
package plugin
