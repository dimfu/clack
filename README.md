# clack

yes it is a metronome to help me play guitar. instead of using the available metronome apps im just gonna build it my self, what a sigma.

## usage

```bash
# install the dependencies
go mod download

# build the binary
go install .

# run the thing
clack --tempo=120 -timesig="4/4"
`````