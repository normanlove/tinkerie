# Signing Tinkerie.exe with SignPath

Windows shows "Windows protected your PC" for programs that aren't signed.
The SignPath Foundation signs open-source programs for free, using a
certificate issued to the foundation, after checking that GitHub built the
file from this repository. The code signing policy is on the website
(`docs/code-signing.html`) and in the main README.

Until SignPath is set up, releases simply come out unsigned: the signing
steps in `.github/workflows/release.yml` are skipped while the
`SIGNPATH_ORGANIZATION_ID` variable is missing.

## Steps for the maintainer (nrmluvdev)

1. **Turn on two-factor authentication** on your GitHub account
   (Settings › Password and authentication). SignPath requires it for
   everyone on the project, on GitHub and on SignPath.
2. **Apply** at <https://signpath.org/apply>. Useful answers:
   - Project: Tinkerie, https://github.com/normanlove/tinkerie
   - Website / download page: https://tinkerie.net/
   - Code signing policy: https://tinkerie.net/code-signing.html
   - Licence: GPL-3.0-or-later (all of Tinkerie's own code; bundled parts are
     listed in apps/scopie/THIRD-PARTY-NOTICES.md and other notices)
   - What is signed: one file, Tinkerie.exe, built by GitHub Actions on
     GitHub-hosted runners (`.github/workflows/release.yml`)
   - Team: one person, nrmluvdev (GitHub: normanlove), who is author,
     reviewer and approver
3. **When approved**, in SignPath.io:
   - Add the predefined **GitHub.com** trusted build system to the
     organisation and link it to the Tinkerie project.
   - Install the **SignPath GitHub App** for this repository.
   - Make the project slug `tinkerie` and the signing policy slug
     `release-signing` (or change the slugs in release.yml to match).
   - Paste `.signpath/artifact-configuration.xml` in as the project's
     default artifact configuration.
   - Make yourself the policy's **approver**, so every release waits for
     you to approve it (SignPath requires manual approval).
   - Create an API token for a user with *submitter* rights.
4. **In GitHub** (repo › Settings › Secrets and variables › Actions):
   - Secret `SIGNPATH_API_TOKEN`: the API token.
   - Variable `SIGNPATH_ORGANIZATION_ID`: your SignPath organisation ID.

From then on each release pauses at "Sign Tinkerie.exe" until you approve
the signing request in SignPath (you get an email; there are 5 hours).
The signed Tinkerie.exe then goes into the release on its own and inside the
Windows zip.
