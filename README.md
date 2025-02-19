# imuachain

Imuachain is an omnichain restaking protocol that combines the crypto-economic security from multiple blockchain networks
and tokens, and extends it to any off-chain system. The protocol is designed with a modular architecture, incorporating
Tendermint-based Byzantine Fault Tolerant (BFT) consensus mechanism, Zero-Knowledge (ZK) light-client bridging, and a
fully EVM-compatible execution environment. This design enables smooth interactions for restakers and seamless
integration for developers. Additionally, we introduce novel concepts, such as Union Restaking, where off-chain services
can form a union to extend the crypto-economic security of their own tokens to each other. By pooling crypto-economic
security and extending it to off-chain systems, Imuachain powers an open market for decentralized trust.

## Documentation

To learn how Imuachain works from a high-level perspective, see the [Imuachain Whitepaper](https://t.co/A4y4YcOuEC)

## Creating docker images

1. Once the dependencies are installed, execute
`make localnet-init`, this will generate the cluster configuration file.
2. Run the following command to create the docker image:

    ```bash
    make localnet-build
    # Check if images build done
    docker images
    ```

3. Launch the chain node:

    ```bash
    make localnet-start
    # Check if containers are all up
    docker ps
    ```

## Interacting with a local node

With a node running, the imuad binary can be used to interact with the node. Run `./bin/imuad <command> --help` to
get information about the available commands.
