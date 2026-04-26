# Changelog

## [1.1.0](https://github.com/gofhir/ecl/compare/v1.0.0...v1.1.0) (2026-04-26)


### Features

* add gofhir-ecl CLI and ECL v2.2 conformance suite ([4544b17](https://github.com/gofhir/ecl/commit/4544b17ada99709390f103212291260d263564f1))
* gofhir-ecl CLI and ECL v2.2 conformance suite ([d194a9b](https://github.com/gofhir/ecl/commit/d194a9b38be20c75540ef2d48ea47ec16dd84ecb))

## [1.0.0](https://github.com/gofhir/ecl/compare/v0.1.0...v1.0.0) (2026-04-25)


### ⚠ BREAKING CHANGES

* **ecl:** complete v2.2 (match-type, multi-value filters, ^R, MemberOf projection)

### Features

* **ecl:** complete v2.2 (match-type, multi-value filters, ^R, MemberOf projection) ([589ebc6](https://github.com/gofhir/ecl/commit/589ebc6e5acb33bbf8e7af9cb42a7c3b95217b56))
* **ecl:** extend Relationship struct and DataProvider interface for v2.2 features ([2309deb](https://github.com/gofhir/ecl/commit/2309debda9d5067c0af93bc014af65842be84105))
* **ecl:** implement AltIdentifier resolution via DataProvider ([428a430](https://github.com/gofhir/ecl/commit/428a4301a603ba7ec90e9a497ee3732a70241007))
* **ecl:** implement attribute cardinality [min..max] for ungrouped and grouped refinements ([3139200](https://github.com/gofhir/ecl/commit/3139200f88953e653e4fd2765981f5880806aaad))
* **ecl:** implement concrete value comparisons inside grouped refinements ([c800685](https://github.com/gofhir/ecl/commit/c8006855c92af31f5c57d205627abc8b36b0ef89))
* **ecl:** implement dialect filter support ([6c4cdba](https://github.com/gofhir/ecl/commit/6c4cdbacdd2179ddbf1653c7edeacd71ee8548d9))
* **ecl:** implement member field filter projection ([3325c5b](https://github.com/gofhir/ecl/commit/3325c5b0e87ba2de4869e9d05afa2949ceec467b))
* **ecl:** implement negated filter operators (!=) for term, type, language, definitionStatus, module ([cb134cd](https://github.com/gofhir/ecl/commit/cb134cd5a48535762514267c4ef6ccdc12a91f75))
* **ecl:** implement reverse attribute inside grouped refinements ([adaf158](https://github.com/gofhir/ecl/commit/adaf15892884716e0afac397ef7baa50e69b8e2e))
* **ecl:** implement reverse attribute with concrete value ([265fbb9](https://github.com/gofhir/ecl/commit/265fbb9049d3122429994416c710477a74b30be0))
* **ecl:** implement reverse attribute with wildcard value (R attr = *) ([96f4d6e](https://github.com/gofhir/ecl/commit/96f4d6e53714e6b554a7b5c31b03737ee2f1e566))
* **ecl:** implement string and boolean concrete value comparisons ([275e4ac](https://github.com/gofhir/ecl/commit/275e4ac6450565ee2ef011e2498fb8c3186c80fd))
