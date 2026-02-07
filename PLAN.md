# Plan

- [ ] Do we have enough tests (AAA pattern) so that we avoid breaking
      functionality?
  - [ ] Measure code coverage and inspect non-covered parts and whether we
        should cover with tests.
  - [ ] Can we create complete test scenarios with composed Config.Auto setups?
  - [ ] Can we end-to-end test the bootstrapper (`pocket init`)?
- [x] Add [zensical](https://github.com/zensical/zensical) as tool into the
      "tools" package, with flags -serve and -build. For this tool, we want to
      maintain the version string in Pocket (with Renovate-ability to bump), and
      it would be ideal if we simply abstract away the whole Python setup and
      need of a .venv from projects.
- [ ] Analyze Pocket
  - [ ] DX - do we have good developer experience?
  - [ ] From a DX perspective; is the API surface easy to understand?
  - [ ] Long-term maintainability, is the codebase simple and ideomatic to Go?
  - [ ] From a files/packages perspective; is the git project laid out well?
  - [ ] From a Go ideomatic view; is the project following Go ideoms, leveraging
        std lib, easy to understand?
  - [ ] Compare with pocket-v1 (~/code/public/pocket-v1); which areas have been
        improved, which areas were done better/simpler in pocket-v1?
- [ ] Verify that all documentation is up to date. We need to mention all public
      API methods and configurable parts.
- [ ] We are in documentation sometimes distinguishing "end-users" and
      "task/tool authors". These are for the most part going to be the same
      person. The difference is really that one person might build tasks/tools
      and then reuse them in multiple projects. We should make that more clear
      in the different markdown documentation files we have.
- [ ] We keep documentation on reference, architecture, user guides. Is there
      overlap and/or risk of updating one but not the other and then cause a
      drift where the documentation is not aligned? Can we consolidate the
      documentation better, and here I'm actually thinking alot about LLMs
      finding one place where documentation needs updating and doesn't see the
      other file which also needs updating.
