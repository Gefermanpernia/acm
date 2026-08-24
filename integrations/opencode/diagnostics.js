const maxEvents = 256;
const maxBytes = 64 << 10;
const maxText = 32;

function boundedText(value) {
  return typeof value === "string" ? value.slice(0, maxText) : "unknown";
}

export function redactDiagnostic(event) {
  return {
    time: Number.isFinite(event?.time) ? event.time : 0,
    component: boundedText(event?.component),
    event: boundedText(event?.event),
    outcome: boundedText(event?.outcome),
    retryable: event?.retryable === true,
  };
}

export function boundedDiagnostics(events) {
  const result = events.slice(-maxEvents).map(redactDiagnostic);
  while (Buffer.byteLength(JSON.stringify(result)) > maxBytes) {
    result.shift();
  }
  return result;
}
