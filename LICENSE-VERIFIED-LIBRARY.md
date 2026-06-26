# Agentic SRE — Verified Signature Library License

**Version 1.0 (DRAFT) — 2026-05-22**
**Licensor:** Srenix, Inc. ("Bionic")
**Contact:** licensing@baisoln.com

> ## ⚠️ DRAFT — DO NOT EXECUTE WITHOUT LEGAL REVIEW
>
> **This document is a non-binding draft.** It was authored by an
> engineering team to capture Bionic's *intent* for the commercial
> Verified Signature Library license; **it has not been reviewed by
> licensed counsel** in any jurisdiction. Specific risks an engineering
> draft cannot evaluate:
>
> - **Enforceability of the use restrictions** (Section 4) under
>   competition / antitrust law in the EU and certain US states.
> - **Validity of the warranty disclaimer and liability cap**
>   (Sections 9–10) — consumer-protection law in some jurisdictions
>   voids or limits these clauses, especially for SaaS or "essential
>   facility" software.
> - **The Delaware forum-selection and governing-law choice**
>   (Section 12) — non-US licensees may have a statutory right to a
>   local forum that overrides this clause.
> - **The "competing product" prohibition** (Section 4.5) — overly
>   broad as drafted; may not survive scrutiny.
> - **The trial-use evaluation period** (Section 6) — needs to align
>   with the subscription agreement's commercial structure.
>
> **Before the first paid subscription is offered for sale, Bionic
> MUST:**
>
> 1. Have this document reviewed by counsel licensed in Delaware (the
>    chosen governing-law jurisdiction).
> 2. Have it reviewed by counsel in each customer-target jurisdiction
>    where Bionic intends to sell (minimum: a US-wide SaaS attorney
>    + EU / UK / India coverage if those markets are targeted).
> 3. Reconcile it with the underlying subscription agreement
>    (master services agreement) so the documents don't contradict
>    each other on payment terms, termination, or auto-renewal.
> 4. Insert Bionic's actual registered legal-entity name and address
>    (line 4 above references "Srenix, Inc." — confirm
>    incorporation status before use).
> 5. Add a signature block or specify the electronic-acceptance
>    mechanism (click-through, DocuSign, etc.).
>
> Until items 1-5 are complete, this file's content is **informational
> only** and is not part of any binding agreement between Bionic and
> any user of the Verified Signature Library.

---

## 1. What this document covers

The Agentic SRE ("Srenix") project is dual-licensed:

| Component | License | Where it lives |
|---|---|---|
| Srenix engine (the `srenix` binary, all of `cmd/`, `internal/`, `pkg/`) | **Apache License 2.0** | This repository, [`LICENSE`](LICENSE) |
| Default Signature Library — the analyzers, fixers, probes, and rule-based investigator that ship in [`catalog/`](catalog/) of this repository | **Apache License 2.0** | This repository, covered by [`LICENSE`](LICENSE) |
| **Verified Signature Library** ("the Library") — the curated, regression-tested, monthly-updated bundle of additional analyzers, fixers, probes, investigators, and AI prompts distributed with the paid Srenix Enterprise tier | **This document (commercial subscription license)** | Distributed as a signed bundle to subscribers only; not in this repository |

**Apache 2.0 is unaffected.** Nothing in this document modifies or supersedes the Apache 2.0 license that covers the Srenix engine and the Default Signature Library. You may continue to use, modify, and redistribute the engine and Default Library under Apache 2.0 terms regardless of whether you are a Verified Signature Library subscriber.

---

## 2. Definitions

- **"Library"** means the Verified Signature Library — the bundle of files distributed by Srenix to active subscribers, signed by Bionic's release-signing key, containing curated detector signatures, fixer recipes, investigator rules, and prompt templates not present in the public Apache-2.0 repository.
- **"Engine"** means the Srenix binary and its Default Signature Library, distributed under Apache 2.0.
- **"Subscriber"** means a legal entity that has executed a current, paid Srenix Enterprise subscription agreement with Bionic and is not in breach of that agreement.
- **"Subscription Period"** means the time during which the Subscriber's Srenix Enterprise subscription is paid and active.
- **"Production Use"** means use of the Library to monitor or remediate any Kubernetes cluster that is not solely a development, test, or non-production environment.
- **"Internal Use"** means use within the Subscriber's own organization, on infrastructure the Subscriber owns or has direct contractual access to, on behalf of the Subscriber's own business operations. Internal Use does not include offering the Library — directly or indirectly — as a service to third parties.

---

## 3. License grant

Subject to the Subscriber's continued compliance with this license and the underlying subscription agreement, Bionic grants the Subscriber a **non-exclusive, non-transferable, non-sublicensable, worldwide, royalty-free** license, **for the duration of the Subscription Period only**, to:

1. **Install** the Library bundle alongside the Engine on infrastructure the Subscriber owns or controls.
2. **Use** the Library, in combination with the Engine, for Internal Use.
3. **Make a reasonable number of copies** of the Library bundle for backup, disaster recovery, and rollback purposes.
4. **Modify configuration files** (Helm values, runtime flags) that adjust how the Library's signatures are applied to the Subscriber's clusters. This permission does not extend to modifying the signature definitions themselves.

---

## 4. Restrictions

The Subscriber shall not, and shall not permit any third party to:

1. **Redistribute** the Library, in whole or in part, in any form, to any party that does not hold its own current Srenix Enterprise subscription.
2. **Offer the Library as a service** — including but not limited to managed Kubernetes services, SaaS platforms, consulting engagements, or any other arrangement where third parties derive the benefit of the Library without holding their own subscription.
3. **Reverse engineer, decompile, or disassemble** the Library, except to the extent that applicable law expressly permits such activity notwithstanding this restriction.
4. **Extract, copy, or republish** the signature definitions, rules, or prompts from the Library into any other tool, product, or open-source project.
5. **Use the Library to develop a competing product** — meaning any tool, service, or platform whose primary function is automated detection or remediation of Kubernetes cluster health issues.
6. **Remove or modify** any copyright, license, signature, or attribution notice present in the Library bundle.
7. **Continue using** the Library after the Subscription Period ends or the subscription is terminated for breach.

Use of the Engine and the Default Signature Library under Apache 2.0 is **not** restricted by this section. The above restrictions apply solely to the Verified Signature Library.

---

## 5. Updates and subscription mechanics

5.1. **Monthly cadence.** Bionic publishes new versions of the Library on a target cadence of one release per calendar month. Bionic makes no warranty that any given month will produce a release.

5.2. **Subscriber receives updates** during the Subscription Period at no additional charge.

5.3. **Pinned versions.** The Subscriber may, at their discretion, run an older version of the Library than the latest available. Bionic provides no security maintenance or back-porting for Library versions older than the three most recent releases.

5.4. **Signature deprecation.** Bionic may, in any release, mark previously shipped signatures as deprecated or remove them. When a signature is removed, Bionic will document the removal in the Library bundle's `CHANGELOG.md` and where possible offer guidance on replacement signatures.

5.5. **Cluster scope.** The subscription covers use across the Subscriber's entire fleet of Kubernetes clusters within the Subscriber's organization. There is no per-cluster, per-node, or per-pod metering at this license tier.

---

## 6. Trial use

Bionic may, from time to time, distribute time-limited evaluation copies of the Library to prospective subscribers. Evaluation use is permitted under this license **for the evaluation period stated in the bundle's manifest** (typically 30 days), for non-production evaluation only. Evaluation copies may not be used in Production Use and must be removed when the evaluation period ends.

---

## 7. Ownership

Bionic retains all right, title, and interest in and to the Library, including all intellectual property rights. This license grants only the rights expressly stated in Section 3 and creates no other ownership or licensing rights.

---

## 8. Feedback

If the Subscriber provides Bionic with feedback, suggestions, bug reports, or improvement ideas regarding the Library, the Subscriber grants Bionic a perpetual, royalty-free, worldwide license to use that feedback for any purpose, without obligation to the Subscriber. Feedback does not become a derivative work of the Library.

---

## 9. No warranty

THE LIBRARY IS PROVIDED **"AS IS"** AND **"AS AVAILABLE"** WITHOUT WARRANTY OF ANY KIND, WHETHER EXPRESS, IMPLIED, OR STATUTORY, INCLUDING BUT NOT LIMITED TO WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, TITLE, OR NON-INFRINGEMENT.

**Operational scope.** Srenix — both the Engine and the Library — performs automated detection and, where configured, automated remediation of Kubernetes cluster state. Automated remediation carries inherent risk. The Subscriber acknowledges that:

- No signature, however well-tested, can anticipate every cluster configuration.
- Bionic does not guarantee that any signature will detect every instance of the condition it targets, nor that any fixer will succeed without unintended side effects.
- The Subscriber is responsible for configuring Srenix's safety controls (dry-run mode, fixer allow-lists, GitOps guards, approval gates) appropriately for their environment.

Bionic strongly recommends that all new signatures and fixers be evaluated in a non-production environment before being enabled in production.

---

## 10. Limitation of liability

TO THE MAXIMUM EXTENT PERMITTED BY APPLICABLE LAW:

10.1. **Cap on direct damages.** Bionic's total cumulative liability for any claim arising out of or relating to this license shall not exceed the total subscription fees actually paid by the Subscriber to Bionic during the twelve (12) months immediately preceding the event giving rise to the claim.

10.2. **No consequential damages.** Bionic shall not be liable for any indirect, incidental, consequential, special, exemplary, or punitive damages, including but not limited to lost profits, lost revenue, lost data, business interruption, or loss of goodwill, even if Bionic has been advised of the possibility of such damages.

10.3. **Carve-outs.** The limitations in this Section 10 do not apply to (a) the Subscriber's breach of Section 4 (Restrictions), (b) either party's indemnification obligations under the subscription agreement, or (c) liability that cannot be excluded under applicable law.

---

## 11. Termination

11.1. **Termination for non-payment or breach.** This license terminates automatically and immediately upon (a) expiration of the Subscription Period without renewal, (b) the Subscriber's failure to pay any subscription fee within thirty (30) days after it becomes due, or (c) the Subscriber's material breach of Section 4 (Restrictions) that is not cured within thirty (30) days after Bionic provides written notice.

11.2. **Effect of termination.** On termination, the Subscriber shall (a) immediately stop using the Library, (b) delete all copies of the Library from all systems within fourteen (14) days, and (c) certify deletion in writing on Bionic's request.

11.3. **Survival.** Sections 4 (Restrictions, post-termination only), 7 (Ownership), 8 (Feedback), 9 (No warranty), 10 (Limitation of liability), 11 (Termination), and 12 (Governing law) survive termination.

11.4. **Engine unaffected.** Termination of this license does **not** affect the Subscriber's rights to use the Engine and the Default Signature Library under Apache 2.0. The Subscriber may continue to use the Apache-2.0-licensed portions of Srenix after the subscription ends; only the Library bundle must be removed.

---

## 12. Governing law and dispute resolution

12.1. **Governing law.** This license is governed by the laws of the State of Delaware, United States, without regard to its conflict-of-laws principles. The United Nations Convention on Contracts for the International Sale of Goods does not apply.

12.2. **Forum.** Any dispute arising out of this license shall be brought exclusively in the state or federal courts located in Wilmington, Delaware. Both parties consent to personal jurisdiction in those courts.

12.3. **Equitable relief.** Bionic may seek injunctive or other equitable relief in any court of competent jurisdiction to protect its intellectual property rights, without posting bond and without the requirement to first exhaust other remedies.

---

## 13. General

13.1. **Entire agreement.** This document, together with the executed subscription agreement, constitutes the entire agreement between the parties regarding the Library and supersedes any prior or contemporaneous representations.

13.2. **No assignment.** The Subscriber may not assign or transfer this license, by operation of law or otherwise, without Bionic's prior written consent. Bionic may assign this license freely.

13.3. **Severability.** If any provision of this license is held unenforceable, the remainder remains in effect.

13.4. **No waiver.** Bionic's failure to enforce any provision is not a waiver of its right to do so later.

13.5. **Notices.** Notices to Bionic shall be sent to `licensing@baisoln.com`. Notices to the Subscriber shall be sent to the email address on the subscription agreement.

13.6. **Changes to this license.** Bionic may update this license for new releases of the Library. Continued use of new Library releases after the update constitutes acceptance. The version of this license in effect for a given Library bundle is the version shipped inside that bundle.

---

## Contact

- **Licensing inquiries:** licensing@baisoln.com
- **Security disclosures:** security@srenix.ai — see [SECURITY.md](SECURITY.md)
- **Open-source engine support:** GitHub Issues at https://github.com/srenix-ai/agentic-sre

---

*Copyright © 2026 Srenix, Inc. All rights reserved. The Srenix engine and Default Signature Library are licensed under Apache License 2.0. The Verified Signature Library is proprietary and licensed separately under the terms above.*
