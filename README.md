# vaniteth
Generates Ethereum vanity addresses

By default, generates vanity account addresses, using a scoring function that prioritises smaller addresses, which produces addresses with many leading zeroes. You can choose the scoring function with `--scorer`; valid options include:

 - `least`: Scores smaller addresses more highly
 - `most`: Scores larger addresses more highly
 - `ascending`: Scores addresses on the length of ascending sequences (11122579...)
 - `strictAscending`: Scores addresses on the length of ascending sequences, with no gaps permitted (1122344...)

Pull requests for more scoring functions are most welcome.

To generate contract addresses instead of account addresses, supply the flag `--contract`. The flag `--maxnonce` accepts an integer value for the maximum nonce (number of sent transactions) from the owning address to search for. Generating new addresses for an account is much faster than generating a new account, so searching many nonces is quicker. However, an address with a large nonce will require sending a lot of dummy transactions from the owning account before sending the desired one. The default value is 32.

Output is a newline separated list of mined addresses, nonces (if in contract mode) and raw private keys.

A docker image for this binary is available on dockerhub [here](https://hub.docker.com/r/arachnid/vaniteth/).
