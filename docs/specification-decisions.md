# JSON:API specification decisions

This register records observable choices for JSON:API 1.1, its official
extensions, profiles, recommendations, and referenced standards. The typed
authority is [`specification/decisions.json`](../specification/decisions.json);
this document preserves the same values for human review.

Resolved decisions are compatibility-sensitive. Unresolved decisions block
repository and release verification. Superseded decisions remain here with a
link to their replacement.

## JSONAPI-DEC-001: Non-compliant and unrecognized members
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** ambiguity
- **Decision scope:** normative
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Document Structure and @-Members
- **Requirement strength:** MUST
- **Issue:** JSON:API forbids additional members in defined objects while requiring implementations to ignore non-compliant members.
- **Credible interpretations:**
  - Reject every unknown member.
  - Ignore non-compliant members while validating recognized content.
- **Known peer behavior:** No maintained peer fixture is pinned; the normative format controls this behavior.
- **Selected behavior:** Core and Atomic decoders ignore unrecognized members while continuing to validate recognized content.
- **Rationale:** Ignoring non-compliant members preserves forward compatibility without weakening recognized-member validation.
- **Security consequences:** Ignored members cannot satisfy required structure or activate application semantics.
- **Resource consequences:** Unknown-member scanning remains subject to configured byte, depth, member, and value limits.
- **Compatibility consequences:** New unknown members do not break otherwise valid documents.
- **Wire consequences:** Ignored members are not emitted when the document is marshaled again.
- **Executable evidence:** TestDecodersIgnoreNonCompliantMembers, TestAtAndNonCompliantMembersAreIgnored, TestCoreCodecIgnoresUnregisteredExtensionMember
- **Fixture evidence:** unknown_member_test.go
- **Fuzz evidence:** FuzzUnmarshal
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `Unmarshal`, `UnmarshalWith`, `Codec.Unmarshal`, and `UnmarshalAtomic`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** The requirement is present in the JSON:API 1.1 format; no erratum is required.
- **Reconsider when:** A later JSON:API version changes unknown-member handling.

## JSONAPI-DEC-002: Extension namespaces and profile authority
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** omission
- **Decision scope:** defensive
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Extensions, Extension Members, Profiles, and Content Negotiation
- **Requirement strength:** not specified
- **Issue:** JSON:API defines extension and profile authority but does not prescribe a safe runtime registration mechanism.
- **Credible interpretations:**
  - Activate semantics from every declared URI.
  - Require explicit immutable registration before activating semantics.
- **Known peer behavior:** No maintained peer matrix is pinned across all member scopes.
- **Selected behavior:** Extension members and profile validators become semantic only through explicit immutable registration.
- **Rationale:** Registration prevents untrusted declarations from activating code or weakening core validation.
- **Security consequences:** Untrusted extension and profile declarations cannot activate callbacks implicitly.
- **Resource consequences:** Registered callback execution remains bounded by codec and document limits.
- **Compatibility consequences:** Unknown profiles remain declarations and unknown extension members follow JSONAPI-DEC-001.
- **Wire consequences:** Registered members round-trip only at their declared object scope.
- **Executable evidence:** TestNewCodecRejectsInvalidExtensionDefinitions, TestCodecRoundTripsRegisteredResourceExtensionMember, TestCodecAppliesRegisteredProfileDocumentValidation, TestProfileValidatorCannotMutateValidatedDocument
- **Fixture evidence:** member_codec_test.go, profile_codec_test.go
- **Fuzz evidence:** FuzzMemberRegistry
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `NewCodec`, `ExtensionDefinition`, `MemberDefinition`, `ProfileDefinition`, and `NewNegotiator`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** Official extension and profile registries remain the upstream authorities.
- **Reconsider when:** JSON:API defines a safe discovery and activation mechanism.

## JSONAPI-DEC-003: Query parameter families
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** omission
- **Decision scope:** application-policy
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Fetching Data and Query Parameters
- **Requirement strength:** not specified
- **Issue:** JSON:API reserves query families but leaves filter, pagination, and implementation-specific semantics application-owned.
- **Credible interpretations:**
  - Preserve every query parameter.
  - Accept only core and explicitly registered implementation or extension families.
- **Known peer behavior:** No maintained router peer is pinned because routing and backend execution are outside this package.
- **Selected behavior:** Parse core families and accept implementation or extension families only through explicit registration.
- **Rationale:** Explicit registration keeps generic parsing separate from application query semantics.
- **Security consequences:** Unknown families cannot silently activate backend behavior.
- **Resource consequences:** Parameter count, name, value, and nesting limits bound parser work.
- **Compatibility consequences:** Registered custom operators remain explicit and local to the configured parser.
- **Wire consequences:** Ordering and explicit empty values are preserved; malformed or unknown families return structured errors.
- **Executable evidence:** TestParseQueryParameters, TestParseQueryPreservesExplicitEmptyIncludeAndSort, TestQueryParserAcceptsRegisteredCustomAndExtensionFamilies, TestParseQueryRejectsMalformedOrUnknownParameters
- **Fixture evidence:** query_test.go
- **Fuzz evidence:** FuzzParseQuery
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `ParseQuery`, `NewQueryParser`, and `QueryParser.Parse`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** JSON:API 1.1 defines no standard filter operator vocabulary.
- **Reconsider when:** A base revision or applied extension standardizes an application-owned family.

## JSONAPI-DEC-004: Relationship objects and linkage
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** ambiguity
- **Decision scope:** normative
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Relationships and Resource Linkage
- **Requirement strength:** MUST
- **Issue:** Relationship objects may contain several qualifying members while linkage has absent, null, to-one, and to-many forms.
- **Credible interpretations:**
  - Require data in every relationship.
  - Accept any qualifying member and preserve the exact linkage form.
- **Known peer behavior:** No maintained peer fixture is pinned across every request and response context.
- **Selected behavior:** Require one qualifying recognized member and preserve absent, null, to-one, and to-many linkage distinctly.
- **Rationale:** The selected behavior follows the relationship object contract without inventing authorization or persistence semantics.
- **Security consequences:** Ignored members cannot satisfy required relationship content or authorize mutation.
- **Resource consequences:** Relationship validation remains bounded by document limits.
- **Compatibility consequences:** Applications retain responsibility for authorization and persistence.
- **Wire consequences:** Absent, null, to-one, empty to-many, and nonempty to-many linkage remain distinct.
- **Executable evidence:** TestMarshalRelationshipDataShapes, TestValidateRelationshipRequestContexts, TestRelationshipIdentifierTraversalClassifiesEveryShape, TestValidateAcceptsToManyCompoundLinkage
- **Fixture evidence:** document_test.go, context_validation_test.go
- **Fuzz evidence:** FuzzConstructedDocumentValidation
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `Relationship`, `RelationshipData`, and validation contexts
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** No known erratum changes the relationship minimum-member rule.
- **Reconsider when:** JSON:API changes relationship membership or contextual linkage rules.

## JSONAPI-DEC-005: Included-resource uniqueness and full linkage
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** ambiguity
- **Decision scope:** normative
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Compound Documents and Sparse Fieldsets
- **Requirement strength:** MUST
- **Issue:** Included resources must be unique and fully linked while identity may use id or lid aliases.
- **Credible interpretations:**
  - Compare only type and server id.
  - Build one alias-aware identity graph and apply the sparse-fieldset exception.
- **Known peer behavior:** No maintained peer graph implementation is pinned; specification-derived fixtures are retained.
- **Selected behavior:** Use an alias-aware identity graph, reject duplicate resources, and require full linkage except for the exact sparse-fieldset exception.
- **Rationale:** One identity graph prevents alternate aliases from bypassing uniqueness and linkage checks.
- **Security consequences:** Alias-aware uniqueness prevents identity confusion and duplicate-resource substitution.
- **Resource consequences:** Graph traversal remains bounded by document resource and relationship limits.
- **Compatibility consequences:** The sparse-fieldset exception applies only when the relevant relationship field was excluded.
- **Wire consequences:** A resource cannot appear twice through alternate id and lid aliases.
- **Executable evidence:** TestValidateLinksIncludedResourceThroughItsLocalIdentity, TestIncludedIdentityFollowsValidationContext, TestValidateWithAllowsSparseFieldsetFullLinkageException
- **Fixture evidence:** testdata/valid/compound-document.json
- **Fuzz evidence:** FuzzConstructedDocumentValidation
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `Document.ValidateWith` and `ValidationOptions`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** No known erratum changes id and lid aliasing for compound documents.
- **Reconsider when:** A future revision changes local identity or full-linkage semantics.

## JSONAPI-DEC-006: Sparse fieldsets
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** omission
- **Decision scope:** application-policy
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Sparse Fieldsets
- **Requirement strength:** not specified
- **Issue:** JSON:API standardizes fieldset syntax but not application schema, authorization, or projection behavior.
- **Credible interpretations:**
  - Project generic maps inside the package.
  - Parse fieldsets and leave schema-aware projection to applications.
- **Known peer behavior:** Framework peers combine parsing and serialization, but no equivalent transport-neutral peer is pinned.
- **Selected behavior:** Parse ordered fields and preserve explicit empty presence while leaving schema validation, authorization, and projection to applications.
- **Rationale:** Generic parsing cannot safely infer an application resource schema.
- **Security consequences:** The package never projects or authorizes unknown application fields.
- **Resource consequences:** Fieldset parsing remains bounded by query limits.
- **Compatibility consequences:** Applications may apply their own typed schema and authorization policy.
- **Wire consequences:** Ordered field names and explicit empty fieldsets are preserved.
- **Executable evidence:** TestParseQueryParameters, TestParseQueryPreservesExplicitEmptyIncludeAndSort, TestValidateWithAllowsSparseFieldsetFullLinkageException
- **Fixture evidence:** query_test.go
- **Fuzz evidence:** FuzzParseQuery
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `Query.Fieldsets`, `QueryParser`, and `ValidationOptions`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** JSON:API defines no application schema registry.
- **Reconsider when:** A typed schema integration supplies an authorization-safe projection contract.

## JSONAPI-DEC-007: Pagination parameters and links
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** optional behavior
- **Decision scope:** extension-specific
- **Specification:** JSON:API Cursor Pagination profile
- **Version:** JSON:API Cursor Pagination profile
- **Source authority:** jsonapi-cursor-source
- **Authoritative URL:** https://jsonapi.org/profiles/ethanresnick/cursor-pagination/
- **Section:** Query Parameters, Links, and Page Meta
- **Requirement strength:** not specified
- **Issue:** Base JSON:API leaves pagination strategy open while the optional Cursor profile defines concrete behavior.
- **Credible interpretations:**
  - Apply cursor rules to every page family.
  - Activate cursor rules only through the profile API.
- **Known peer behavior:** No maintained peer fixture is pinned; the published profile examples are retained as authority evidence.
- **Selected behavior:** Preserve generic page parameters and activate Cursor profile parameters, links, metadata, and errors only through the profile API.
- **Rationale:** Optional profile behavior must not become an implicit base JSON:API requirement.
- **Security consequences:** Cursor values remain opaque and applications retain responsibility for integrity and authorization.
- **Resource consequences:** Page size and cursor parsing are bounded by profile and query limits.
- **Compatibility consequences:** Base users may retain non-cursor pagination strategies.
- **Wire consequences:** The canonical profile URI remains `http://jsonapi.org/profiles/ethanresnick/cursor-pagination/` while its monitored document is retrieved over HTTPS.
- **Executable evidence:** TestParseQueryParameters, TestPaginationLinksRequireCollectionData, TestCursorPaginationParsesProfileParameters, TestValidateCursorPaginationLinks
- **Fixture evidence:** cursor_test.go, cursor_page_test.go
- **Fuzz evidence:** FuzzCursorPaginationQuery
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `Query.Page`, `CursorPagination`, `CursorPage`, and cursor metadata helpers
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** The Cursor Pagination profile remains a separately published optional profile.
- **Reconsider when:** JSON:API standardizes base pagination or the profile publishes an incompatible revision.

## JSONAPI-DEC-008: Atomic operation order and rollback
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** omission
- **Decision scope:** defensive
- **Specification:** JSON:API Atomic Operations extension
- **Version:** JSON:API Atomic Operations extension
- **Source authority:** jsonapi-atomic-source
- **Authoritative URL:** https://jsonapi.org/ext/atomic/
- **Section:** Processing and Operation Objects
- **Requirement strength:** not specified
- **Issue:** Atomic Operations requires ordered all-or-nothing processing but does not define a reusable Go transaction lifecycle.
- **Credible interpretations:**
  - Validate document shape without execution support.
  - Require a caller-supplied transaction lifecycle with bounded failure handling.
- **Known peer behavior:** No maintained cross-language transaction executor fixture is pinned.
- **Selected behavior:** Validate before beginning, apply operations in order, validate results before commit, and attempt rollback exactly once after a post-begin failure.
- **Rationale:** An explicit transaction adapter preserves extension ordering while leaving datastore isolation application-owned.
- **Security consequences:** Validation precedes side effects and callback panics become bounded typed errors.
- **Resource consequences:** Callbacks execute sequentially and cancellation stops further operations.
- **Compatibility consequences:** Applications retain responsibility for durable isolation and commit-unknown handling.
- **Wire consequences:** Results preserve operation order and partial success is never emitted as a valid response.
- **Executable evidence:** TestExecuteAtomicAppliesInOrderAndCommits, TestExecuteAtomicRollsBackAtFirstOperationFailure, TestExecuteAtomicRollsBackCommitFailure, TestExecuteAtomicConvertsApplyPanicAndRollsBack, TestExecuteAtomicStopsAndRollsBackOnCancellation
- **Fixture evidence:** atomic_execute_test.go
- **Fuzz evidence:** FuzzUnmarshalAtomic
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** `ExecuteAtomic`, `AtomicTransactionBeginner`, and `AtomicTransaction`
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** Atomic Operations remains an official extension rather than base format text.
- **Reconsider when:** The extension changes processing order, result correspondence, or atomicity.

## JSONAPI-DEC-009: Recommendation status
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** optional behavior
- **Decision scope:** recommended
- **Specification:** JSON:API recommendations
- **Version:** JSON:API recommendations
- **Source authority:** jsonapi-recommendations-source
- **Authoritative URL:** https://jsonapi.org/recommendations/
- **Section:** Recommendations
- **Requirement strength:** informative
- **Issue:** JSON:API recommendations describe conventions that are not base conformance requirements.
- **Credible interpretations:**
  - Enforce recommendations as protocol validation.
  - Document recommendations without making them package wire requirements.
- **Known peer behavior:** Frameworks enforce local conventions, but those conventions are not portable conformance evidence.
- **Selected behavior:** Document useful recommendations without rejecting a base-compliant document for recommendation differences.
- **Rationale:** Informative guidance cannot be promoted into normative validation silently.
- **Security consequences:** Recommendation-only behavior cannot activate routing, filtering, or method override implicitly.
- **Resource consequences:** Applications remain responsible for bounding any convention-specific backend work.
- **Compatibility consequences:** Applications may add stricter local conventions without changing package conformance.
- **Wire consequences:** Base-compliant documents are accepted regardless of recommendation-only naming or URL conventions.
- **Executable evidence:** TestParseQueryParameters, TestQueryParserAcceptsRegisteredCustomAndExtensionFamilies
- **Fixture evidence:** query_test.go
- **Fuzz evidence:** FuzzParseQuery
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** Documentation and examples; no recommendation-only validator
- **Documentation:** docs/specification-decisions.md, docs/recommendations.md
- **Upstream status:** The recommendations page remains separate from the normative format.
- **Reconsider when:** A recommendation is promoted into normative base, extension, or profile text.

## JSONAPI-DEC-010: Conflicts between authorities
- **Status:** resolved
- **Owner:** jsonapi maintainers
- **Classification:** contradiction
- **Decision scope:** defensive
- **Specification:** JSON:API 1.1
- **Version:** JSON:API 1.1
- **Source authority:** jsonapi-1.1-source
- **Authoritative URL:** https://jsonapi.org/format/
- **Section:** Extensions, Profiles, and Conformance
- **Requirement strength:** not specified
- **Issue:** Base, extension, profile, recommendation, example, and application text have different authority and may conflict.
- **Credible interpretations:**
  - Let the most specific artifact override base behavior.
  - Apply each artifact only within its declared authority and block unresolved contradictions.
- **Known peer behavior:** Peer popularity does not determine normative authority.
- **Selected behavior:** Apply base rules first, constrain extensions and profiles to their authority, treat recommendations as informative, and block unresolved direct contradictions.
- **Rationale:** Separating authorities prevents lower-strength guidance from weakening base conformance.
- **Security consequences:** Lower-authority artifacts cannot disable base validation or resource limits.
- **Resource consequences:** Authority checks add no unbounded runtime work.
- **Compatibility consequences:** Moving behavior between authority classes requires compatibility review.
- **Wire consequences:** Base, extension, profile, recommendation, and application behavior remain separately labeled.
- **Executable evidence:** TestCodecAppliesRegisteredProfileDocumentValidation, TestProfileValidatorCannotMutateValidatedDocument, TestExecuteAtomicAppliesInOrderAndCommits, TestCursorPaginationParsesProfileParameters
- **Fixture evidence:** profile_codec_test.go
- **Fuzz evidence:** FuzzMemberRegistry
- **Interoperability evidence:** specification/interoperability.tsv
- **Public APIs:** All parsing, validation, negotiation, extension, and profile APIs
- **Documentation:** docs/specification-decisions.md
- **Upstream status:** No unresolved direct contradiction is recorded at this revision.
- **Reconsider when:** A governing authority publishes a conflicting erratum or revision.

## Unresolved decisions

No known decision is unresolved at this revision. Maintained peer coverage is
not claimed where a decision records only an official corpus or local contract.
New ambiguities remain unresolved until they receive a stable identifier,
authority analysis, executable evidence, and maintainer disposition.
