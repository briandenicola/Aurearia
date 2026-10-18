# Attribution image extension: CodeQL follow-up

PR #827 at beta `5120bcf34101187eac07d1337ecd63c6a58c4d13` is correctly
blocked by the CodeQL aggregate. The four language analysis jobs passed, but
the aggregate reports two high-severity `go/path-injection` findings on the
thumbnail/medium writes in `services/image_service.go:113-114` (alerts 62/63).
No verifier or security gate is changed.

`NormalizeImageExt` already rejects values outside its five-extension allowlist.
Return the five fixed extension literals instead of the normalized input, so
generated filenames contain no user-derived string. Accepted/rejected inputs,
filename suffixes, primary/card semantics, bytes and ownership are unchanged.
No new API, schema, dependencies or deployment behavior.

Added direct unit coverage: fifteen accepted normalized inputs and ten invalid,
traversal, absolute-path and NUL-containing inputs. Disabling the rejection guard
causes all ten negative cases to fail; restored the source and reran the focused
four-package image/privacy/architecture/OpenAPI selection successfully using
the already prepared Go 1.26.9 SDK, with toolchain downloads disabled.

Exact command from `src/api`:
`go test . ./handlers ./repository ./services -run 'TestNormalizeImageExt|TestAttribution|TestImageHandler|TestResolveAuthorizedMediaPath|TestValidateImageData|TestCoinImagePathInActiveShowcase|TestUserRepository|TestArchitecture|TestRegisteredAPIRoutesAreDocumentedInOpenAPI' -count=1`

Owner explicitly approved one reviewer read, then commit/push the fix.
Independent sign-off and exact-head CodeQL/PR verification remain pending.
The separately deferred software QC audit is still required before merge.
No main merge, release, deployment, installation or local application build.

Original reviewer returned bounded pre-push PASS on staged tree
`37ba97fd24286e40cf906a7895cf98e0b214075e`, parent `5120bcf3`, using exactly
the one approved read. Application/test source is unchanged after that verdict;
only this factual receipt follows it. Commit/push is authorized. Exact-head
CodeQL closure and candidate verification still determine completion.
