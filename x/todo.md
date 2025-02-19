# the module logic to be done

## common

- use evm tx as the only entry for any imua operation
- implement the invariant logic for every module to keep the state security
- setting module parameter needs to be done through governance proposal
- pay attention to each module's state when the EVM transaction fails.
- consider which operations require depositing some imua tokens to maintain security

## deposit

## delegation

- delegateTo and undelegateFrom might also need to be implemented using imua as entry
- Need to check the input parameters and deposit some imua token when register to an operator
- the operator can only be registered once
- delegateTo might require the approval of operator to grant the operator permission for selecting a staking user

## assets

- implement the registration of client chain and assets through the governance proposal instead of setting in the genesis

## withdraw

## reward

- consider storing the reward state in its own module state

## imuaslash

- record the slash states of all operators
- provide the function to deploy slash condition for AVS
- provide the function to approve the slash condition deployed by the AVS(to be decided)
- provide the function to submit slash proof

## AVS opted-in

- record the operator opted-in information
