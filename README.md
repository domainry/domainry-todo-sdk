# Domainry Todo SDK

Public Todo contracts and in-process module boundaries.

This module contains no database implementation. A product composition root
selects a Todo implementation through `modulehost.Factory`; the implementation
creates and owns its Store behind `modulehost.ModuleBinding`.
