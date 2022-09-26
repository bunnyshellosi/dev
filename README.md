# bunnyshell-dev

### Known issues

#### Mutagen

If you already have mutagen installed locally, you might see this error:

```
Error: unable to connect to daemon: client/daemon version mismatch (daemon restart recommended)
```

This happens because `bunnyshell-dev` ships it's own version of mutagen, in order to fix this you will have to stop the current running mutagen daemon by running this command:

```
mutagen daemon stop
```
