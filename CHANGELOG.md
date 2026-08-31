# Changelog

## [1.5.0](https://github.com/gofhir/ecl/compare/v1.4.0...v1.5.0) (2026-08-31)


### Features

* **ecl:** support the description id filter via a capability ([21bfaf7](https://github.com/gofhir/ecl/commit/21bfaf7c1dd7ae36d6ce5fe173f6e858a1789fd3))
* **ecl:** support the description id filter via a capability ([a534171](https://github.com/gofhir/ecl/commit/a5341714a0e2569b431f4dc1d8e1abf3ed858983))
* **ecl:** support the memberOf field projection via a capability ([2e94afc](https://github.com/gofhir/ecl/commit/2e94afcb44e243ed0d942eb2df7a630a8b523783))
* **ecl:** support the memberOf field projection via a capability ([40cba5d](https://github.com/gofhir/ecl/commit/40cba5d1f76a6d40b481f628c5e22bff88e5bc60))


### Bug Fixes

* **ecl:** a member filter's clauses must hold on ONE member row ([4148d46](https://github.com/gofhir/ecl/commit/4148d46a96a60ef66522f30ada0cb6477c2a150c))

## [1.4.0](https://github.com/gofhir/ecl/compare/v1.3.0...v1.4.0) (2026-08-28)


### Features

* **mrcm:** enforce in-group cardinality, and count distinct values ([6438076](https://github.com/gofhir/ecl/commit/6438076b75cf03bdcfa8b755799c006ad4b4c3fa))
* **mrcm:** enforce in-group cardinality, and count distinct values ([da4ee82](https://github.com/gofhir/ecl/commit/da4ee821c119bfb8e8e50b9f062081bfbec8d15a))

## [1.3.0](https://github.com/gofhir/ecl/compare/v1.2.0...v1.3.0) (2026-08-28)


### Features

* **ecl:** make parsing linear, bound the input, and fuzz it ([3bb4f1a](https://github.com/gofhir/ecl/commit/3bb4f1a859106cd81b224d7cdeaa20cf82e2e27e))
* **ecl:** make parsing linear, bound the input, and fuzz it ([294f083](https://github.com/gofhir/ecl/commit/294f08370a4b3f8e21aff40ec933497fa7724d5b))

## [1.2.0](https://github.com/gofhir/ecl/compare/v1.1.0...v1.2.0) (2026-08-27)


### Features

* **ecl:** evaluate filter value sets, and resolve dialect aliases via a capability ([c0f977b](https://github.com/gofhir/ecl/commit/c0f977b6889b6e36873a7df1df3f2d3ba5782402))
* **ecl:** expose providertest with an embedded suite, and add UnimplementedDataProvider ([e8dcc30](https://github.com/gofhir/ecl/commit/e8dcc3004223f90c0f47a00757a7b00a226d42f1))
* **ecl:** offer the deferred work as optional provider capabilities ([e473929](https://github.com/gofhir/ecl/commit/e473929f9e6fb6b4a10583ba7e7edb5d3d3cf44b))
* **ecl:** wire NegatingDescriptionProvider into the evaluator ([4120fb1](https://github.com/gofhir/ecl/commit/4120fb1472fae05a6192d7f8f551e16b20c8b2e1))
* **ecl:** wire the last capability, and have VerifyContract check them ([7f438d9](https://github.com/gofhir/ecl/commit/7f438d946139a0206950f1af259c8308b2c0316f))
* **evaluator:** support "!=" and OR on a reverse attribute inside a group ([1a96a25](https://github.com/gofhir/ecl/commit/1a96a259e727633b5013dd6a69c7589c92d7ffcc))
* **providertest:** split contract conformance from fixture conformance ([b9ddeed](https://github.com/gofhir/ecl/commit/b9ddeedf922ee99eacf33c018190ac4e490c69b1))


### Bug Fixes

* address code review of this branch ([364d465](https://github.com/gofhir/ecl/commit/364d465f513025db83362f00c74c8db58b9c9d71))
* address fourth code review of this branch ([9e01dd8](https://github.com/gofhir/ecl/commit/9e01dd814771c6d7bb2416a33a81d4da6b63e653))
* address second code review of this branch ([c5203f7](https://github.com/gofhir/ecl/commit/c5203f73451a11d96e0525875abe7cff1470b6b4))
* **ci:** make release-please find the previous release, and guard the decision ([9775b2f](https://github.com/gofhir/ecl/commit/9775b2fbd9e191fb8de14891f24ef216a26a3ed0))
* **ci:** make release-please find the previous release, and guard the decision ([9a4f16f](https://github.com/gofhir/ecl/commit/9a4f16f0bedfd660cfe04f0a27d82eb18fb5a1df))
* **ci:** pass --repo to gh in the release workflow ([e8055fd](https://github.com/gofhir/ecl/commit/e8055fdc8a914564694e828bfc534b37f6070d8d))
* **ci:** pass --repo to gh in the release workflow ([bd4ec14](https://github.com/gofhir/ecl/commit/bd4ec14b63eb41ef6ad25ce51bc6897f5ee1557d))
* **cli:** send diagnostics to stderr, exit 0 on -h, and add exit codes ([b0a1abb](https://github.com/gofhir/ecl/commit/b0a1abb6af812693119d615a77772a5dd9dd5aa2))
* **conformance:** correct the direction of history supplements ([b683c84](https://github.com/gofhir/ecl/commit/b683c847139c83eb30fd77cecedfe98c11e02019))
* **conformance:** implement term match as word-prefix instead of substring ([2942e72](https://github.com/gofhir/ecl/commit/2942e727703ffabf112353a8525fb88fcc6f112a))
* **ecl:** correct the silently-wrong evaluator semantics, and verify against sources this project did not write ([6c976f1](https://github.com/gofhir/ecl/commit/6c976f11324cb04f186b80f4ff037091337e9538))
* **ecl:** populate the dialect filter and accept literal member field values ([e519a2b](https://github.com/gofhir/ecl/commit/e519a2bbe3bec655dd057c1e5d09de8cc7af801e))
* **ecl:** reject reverse attributes inside an attribute group ([26b1e75](https://github.com/gofhir/ecl/commit/26b1e7511a6b11fea2de916cab4c252749e44a42))
* **ecl:** specify the DataProvider contract and stop nil Sets from panicking ([4b5590b](https://github.com/gofhir/ecl/commit/4b5590b17ad9e82ab2bbda900910b1476346216a))
* **evaluator:** apply cardinality to attribute groups and concrete values ([050c623](https://github.com/gofhir/ecl/commit/050c623e1361e03c0bb076613df6f6e02ec980ee))
* **evaluator:** compose filter negation per clause instead of one global flag ([9d4425b](https://github.com/gofhir/ecl/commit/9d4425b965915f2e0ebf84bb7aa89ed29104c459))
* **evaluator:** count source concepts for reverse cardinality, per the spec ([01e13de](https://github.com/gofhir/ecl/commit/01e13de65b4002c19c4c4e4b2bdefb7683a1e87d))
* **evaluator:** evaluate OR inside refinements as a union, not an intersection ([e3f7307](https://github.com/gofhir/ecl/commit/e3f73075de4ef18ecd0211a09b6edb3934860b57))
* **evaluator:** make "!=" negate the attribute value, not its existence ([db14f97](https://github.com/gofhir/ecl/commit/db14f973ec8abf4b9bfa3cec2f7a9a2291cf0545))
* **evaluator:** make Top and Bottom use transitive ancestors and descendants ([34576e0](https://github.com/gofhir/ecl/commit/34576e0ed985f6b44f97fdf72aedd6f6f0390692))
* make the reverse attribute path consistent, and stop widening on tokens ([caeefdf](https://github.com/gofhir/ecl/commit/caeefdf3decdef74198b4beff7b9a2b63075e611))
* **mrcm:** treat domain rows as alternatives and check minimum cardinality ([e825e03](https://github.com/gofhir/ecl/commit/e825e036293a9b13c868d2a65682ec38866e9784))
* **parser:** model the filter branches that were being dropped ([38d3924](https://github.com/gofhir/ecl/commit/38d3924d9b9959ee9f0df0a2fb921474a8693308))
* **parser:** reject trailing input and report lexer errors ([ef1a586](https://github.com/gofhir/ecl/commit/ef1a58619d332f50c8576cfd894279709005c204))
* **scg,sctid:** accept juxtaposed groups, default to equivalentTo, enforce partitions ([b7c1916](https://github.com/gofhir/ecl/commit/b7c1916e75a77fb2ce7a7d9feeedcc0ad85dda24))

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
