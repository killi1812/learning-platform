---
trigger: always_on
---

# How to write a commit message

## The seven rules of a great Git commit message

1. Separate subject from body with a blank line
2. Limit the subject line to 50 characters
3. Capitalize the subject line
4. Do not end the subject line with a period
5. Use the imperative mood in the subject line
6. Wrap the body at 72 characters
7. Use the body to explain what and why vs. how

## 2. Limit the subject line to 50 characters

- soft limit at 50 characters but hard limit at 70
- this repo puts the ticket id before the subject, for example `[TCB-12345] Fix ...`.
  Count that prefix against the budget.

## 5. Use the imperative mood in the subject line

A correctly formed Git commit subject line can always complete this sentence:

> If applied, this commit will your subject line here

For example:

> If applied, this commit will refactor subsystem X for readability
> If applied, this commit will update getting started documentation
> If applied, this commit will remove deprecated methods
> If applied, this commit will release version 1.0.0
> If applied, this commit will merge pull request #123 from user/branch

- The non-imperative forms that follow do not complete the sentence:

> If applied, this commit will fixed bug with Y
> If applied, this commit will changing behavior of X
> If applied, this commit will more fixes for broken stuff
> If applied, this commit will sweet new API methods

- _Remember_: The imperative is important only in the subject line. You can relax this restriction in the body.

## 7. Use the body to explain what and why vs. how

This commit from Bitcoin Core is a good example. It explains what changed and why:

```text
commit eb0b56b19017ab5c16c745e6da39c53126924ed6
Author: Pieter Wuille <pieter.wuille@gmail.com>
Date:   Fri Aug 1 22:57:55 2014 +0200

   Simplify serialize.h's exception handling

   Remove the 'state' and 'exceptmask' from serialize.h's stream
   implementations, as well as related methods.

   As exceptmask always included 'failbit', and setstate was always
   called with bits = failbit, all it did was immediately raise an
   exception. Get rid of those variables, and replace the setstate
   with direct exception throwing (which also removes some dead
   code).

   As a result, good() is never reached after a failure (there are
   only 2 calls, one of which is in tests), and can just be replaced
   by !eof().

   fail(), clear(n) and exceptions() are just never called. Delete
   them.
```
