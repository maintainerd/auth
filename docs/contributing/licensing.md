# Licensing Maintainerd Projects

Maintainerd projects are licensed under the Apache License, Version 2.0.
Reyco Seguma is the original creator of the Maintainerd applications.

Use this checklist for every new Maintainerd repository:

1. Add a top-level `LICENSE` containing the unmodified canonical Apache
   License 2.0 text from <https://www.apache.org/licenses/LICENSE-2.0.txt>.
   Keep the `[yyyy] [name of copyright owner]` text in the Appendix unchanged;
   it is an example showing how to apply the license to a work.
2. Add a top-level `NOTICE` using this template:

   ```text
   Maintainerd <Product Name>
   Copyright <year> Reyco Seguma

   Maintainerd <Product Name> was originally created by Reyco Seguma.
   This product is licensed under the Apache License, Version 2.0.
   See the LICENSE file for details.
   ```

3. State the copyright, license, and `NOTICE` link in the README. Name Reyco
   Seguma as the original creator without removing credit for contributors.
4. Use the SPDX identifier `Apache-2.0` in package metadata. For npm projects,
   also set the `author` field to Reyco Seguma.
5. Do not change dependency-license entries in lockfiles. Before a release,
   review the shipped artifact for bundled third-party works and preserve any
   licenses or attribution notices those works require. Add only required or
   genuinely useful attributions to `NOTICE`.
6. When an individual source file is intended to be distributed independently,
   apply the boilerplate notice from the Apache License Appendix using that
   file type's comment syntax.

When the copyright year changes, use a range such as `2026-2027`; do not replace
the original year. Contributors retain rights in their own contributions unless
a separate written agreement says otherwise.

The Apache Software Foundation provides the canonical license and application
guidance at <https://www.apache.org/legal/apply-license>.
