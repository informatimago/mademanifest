Below is a **systematic extraction of all functional and
non-functional requirements** that are **explicitly stated in *“Proof
of Capability V1 ENG.pdf”***, with **section references**.

Per your instruction, the **Engineering Implementation Specification**
is used **only to disambiguate terminology**, not as a source of
additional requirements.

---

## 1. Functional Requirements

These define **what the implementation must do** to be considered correct.

### FR-1: Execute the Golden Test Case exactly

**Reference:** Section 2 – *Scope of This Proof* 

* The implementation **must make exactly one Golden Test Case pass**.
* There is **no additional functional scope** beyond this case.

---

### FR-2: Consume the Golden Test Case as the complete input specification

**Reference:** Section 2 – *Scope of This Proof* 

The Golden Test Case fully defines:

* Input birth data
* Fixed engine parameters
* Expected output for:

  * Astrology
  * Human Design – Personality
  * Human Design – Design
  * Gene Keys Activation Sequence (V1)

No additional inputs are allowed.

---

### FR-3: Produce output that matches the Golden Test Case exactly

**Reference:** Section 3 – *Definition of Success* 

The implementation **must produce output that is byte-for-byte
identical**, including:

* Gates and lines
* Node policy application
* Object ordering
* Numeric values
* JSON structure and serialization

Any deviation → incorrect.

---

### FR-4: Follow node policies exactly as defined

**Reference:** Section 3 – *Definition of Success* 

* Node policies must be applied **exactly as specified**.
* No substitutions or approximations are allowed.

---

### FR-5: Maintain mandatory object ordering

**Reference:** Section 3 – *Definition of Success* 

* Output ordering is **part of correctness**.
* Reordering is considered a defect even if values are correct.

---

### FR-6: Implement only what is explicitly defined in the canonical documents

**Reference:** Section 4 – *Canonical Truth* 

The **only authoritative sources** are:

* Engineering Implementation Specification
* Engineer Onboarding README
* Golden Test Case JSON

Anything not explicitly defined is **out of scope**.

---

### FR-7: Deliver one of the two accepted deliverable forms

**Reference:** Section 7 – *Expected Deliverables* 

Either:

* **Option A:** Working Go implementation + execution showing the Golden Test Case passes
  or
* **Option B:** Test-first implementation where the Golden Test Case passes exactly

No other delivery format is acceptable.

---

### FR-8: Allow observations without behavioral impact

**Reference:** Section 8 – *Observations* 

* Observations may be documented separately
* Observations **must not change behavior**
* Code must still follow the specification exactly

---

## 2. Non-Functional Requirements

These constrain **how** the system must be implemented and evaluated.

---

### NFR-1: Determinism

**Reference:** Section 1 – *Purpose* 

The implementation must be:

* Deterministic
* Reproducible
* Exact
* Fully specification compliant

---

### NFR-2: Binary evaluation (pass/fail only)

**Reference:** Section 3 – *Definition of Success* 

* No partial credit
* No tolerances
* No post-evaluation discussion
* Any deviation = failure

---

### NFR-3: Programming language is fixed to Go

**Reference:** Section 5.1 – *Programming Language and Runtime* 

* Language **must be Go**
* Go version **must match exactly** what is specified in the provided materials
* No alternative languages permitted

---

### NFR-4: Execution environment is part of correctness

**Reference:** Section 5.2 – *Platform and Environment* 

Mandatory environment conditions:

* OS as specified (CI is authoritative)
* Environment variables:

  * `TZ=UTC`
  * `LC_ALL=C`
  * `LANG=C`
* Line endings: LF
* Encoding: UTF-8

Deviation invalidates correctness.

---

### NFR-5: Dependency constraints are absolute

**Reference:** Section 5.3 – *Dependencies* 

* Swiss Ephemeris:

  * Exact version
  * Exact data files
  * Exact checksums
* Only explicitly permitted libraries may be used
* No alternative math, time, or ephemeris libraries allowed

---

### NFR-6: Floating-point behavior is strictly constrained

**Reference:** Section 5.4 – *Floating Point and Formatting* 

* Calculations must follow the specification exactly
* Rounding only where explicitly allowed
* No implicit normalization
* No “close enough” logic

---

### NFR-7: Interface is fixed and non-extensible

**Reference:** Section 5.5 – *Interface* 

* Input/output contract is fixed
* No alternative interfaces
* No CLI variants
* No HTTP APIs
* No abstraction layers beyond what is specified

---

### NFR-8: No clarification process

**Reference:** Section 12 – *Clarifications* 

* No clarification questions will be answered
* Ambiguity must be resolved by **exact execution**, not interpretation

---

### NFR-9: Explicit exclusions

**Reference:** Section 9 – *Explicitly Out of Scope* 

The implementation must **not include**:

* UI
* Interpretation
* Optimization
* Alternative assumptions
* Rule simplification
* Refactoring for cleanliness

---

### NFR-10: Evaluation ignores style and architecture

**Reference:** Section 11 – *Evaluation Criteria* 

Not evaluated:

* Programming style
* Framework choice
* Architecture preferences
* Performance
* Creativity

Only correctness and compliance matter.

---

## 3. Summary Table

| Category                     | Count |
| ---------------------------- | ----- |
| Functional requirements      | 8     |
| Non-functional requirements  | 10    |
| Total extracted requirements | 18    |

---

If you want, next we can:

* Cross-map these requirements to **testable acceptance criteria**, or
* Produce a **requirements-to-implementation checklist** suitable for
  final self-validation before submission.
