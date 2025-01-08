pragma solidity >=0.8.17;

/// @dev The avs-manager contract's address.
address constant AVSMANAGER_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000901;

/// @dev The avs-manager contract's instance.
IAVSManager constant AVSMANAGER_CONTRACT = IAVSManager(AVSMANAGER_PRECOMPILE_ADDRESS);
/// @author Exocore Team
/// @title AVS-Manager Precompile Contract
/// @dev The interface through which solidity contracts will interact with AVS-Manager
/// @custom:address 0x0000000000000000000000000000000000000901

interface IAVSManager {
    // note:string and bytes will be hashed. address / uintX will not be hashed when using indexed.
    event AVSRegistered(address indexed avsAddr, string sender, string avsName);
    event AVSUpdated(address indexed avsAddr, string sender, string avsName);
    event AVSDeregistered(address indexed avsAddr, string sender, string avsName);
    event OperatorJoined(address indexed avsAddr, string sender);
    event OperatorLeft(address indexed avsAddr, string sender);
    event TaskCreated(
        address indexed taskContractAddress,
        uint64 indexed taskId,
        string sender,
        string name,
        bytes hash,
        uint64 taskResponsePeriod,
        uint64 taskChallengePeriod,
        uint64 thresholdPercentage,
        uint64 taskStatisticalPeriod
    );
    event ChallengeInitiated(
        string sender, bytes taskHash, uint64 taskID, bytes taskResponseHash, string operatorAddress
    );
    event PublicKeyRegistered(string sender, string name);
    event TaskSubmittedByOperator(
        address indexed taskContractAddress,
        uint64 indexed taskId,
        string sender,
        bytes taskResponse,
        bytes blsSignature,
        uint8 phase
    );

    /// @dev Register AVS contract to EXO.
    /// @param sender The external address for calling this method.
    /// @param avsName The name of AVS.
    /// @param minStakeAmount The minimum amount of funds staked by each operator.
    /// @param taskAddr The task address of AVS.
    /// @param slashAddr The slash address of AVS.
    /// @param rewardAddr The reward address of AVS.
    /// @param avsOwnerAddress The owners who have permission for AVS.
    /// @param whitelistAddress The whitelist address of the operator.
    /// @param assetIds The basic asset information of AVS.
    /// @param avsUnbondingPeriod The unbonding duration of AVS.
    /// @param minSelfDelegation The minimum delegation amount for an operator.
    /// @param epochIdentifier The AVS epoch identifier.
    /// @param params 1.miniOptInOperators The minimum number of opt-in operators.
    ///2.minTotalStakeAmount The minimum total amount of stake by all operators.
    ///3.avsReward The proportion of reward for AVS.
    ///4.avsSlash The proportion of slash for AVS.
    function registerAVS(
        address sender,
        string memory avsName,
        uint64 minStakeAmount,
        address taskAddr,
        address slashAddr,
        address rewardAddr,
        address[] memory avsOwnerAddress,
        address[] memory whitelistAddress,
        string[] memory assetIds,
        uint64 avsUnbondingPeriod,
        uint64 minSelfDelegation,
        string memory epochIdentifier,
        uint64[] memory params
    ) external returns (bool success);

    /// @dev Update AVS info to EXO.
    /// @param sender The external address for calling this method.
    /// @param avsName The name of AVS.
    /// @param minStakeAmount The minimum amount of funds staked by each operator.
    /// @param taskAddr The task address of AVS.
    /// @param slashAddr The slash address of AVS.
    /// @param rewardAddr The reward address of AVS.
    /// @param avsOwnerAddress The owners who have permission for AVS.
    /// @param whitelistAddress The whitelist address of the operator.
    /// @param assetIds The basic asset information of AVS.
    /// @param avsUnbondingPeriod The unbonding duration of AVS.
    /// @param minSelfDelegation The minimum delegation amount for an operator.
    /// @param epochIdentifier The AVS epoch identifier.
    /// @param params 1.miniOptInOperators The minimum number of opt-in operators.
    ///2.minTotalStakeAmount The minimum total amount of stake by all operators.
    ///3.avsReward The proportion of reward for AVS.
    ///4.avsSlash The proportion of slash for AVS.
    function updateAVS(
        address sender,
        string memory avsName,
        uint64 minStakeAmount,
        address taskAddr,
        address slashAddr,
        address rewardAddr,
        address[] memory avsOwnerAddress,
        address[] memory whitelistAddress,
        string[] memory assetIds,
        uint64 avsUnbondingPeriod,
        uint64 minSelfDelegation,
        string memory epochIdentifier,
        uint64[] memory params
    ) external returns (bool success);

    /// @dev Deregister avs from exo
    /// @param sender The external address for calling this method.
    /// @param avsName The name of AVS.
    function deregisterAVS(address sender, string memory avsName) external returns (bool success);

    /// @dev RegisterOperatorToAVS operator opt in current avs
    /// @param sender The external address for calling this method.
    function registerOperatorToAVS(address sender) external returns (bool success);

    /// @dev DeregisterOperatorFromAVS operator opt out current avs
    /// @param sender The external address for calling this method.
    function deregisterOperatorFromAVS(address sender) external returns (bool success);

    /// @dev CreateTask , avs owner create a new task
    /// @param sender The external address for calling this method.
    /// @param name The name of the task.
    /// @param hash The data supplied by the contract, usually ABI-encoded.
    /// @param taskResponsePeriod The deadline for task response.
    /// @param taskChallengePeriod The challenge period for the task.
    /// @param thresholdPercentage The signature threshold percentage.
    /// @param taskStatisticalPeriod The statistical period for the task.
    function createTask(
        address sender,
        string memory name,
        bytes calldata hash,
        uint64 taskResponsePeriod,
        uint64 taskChallengePeriod,
        uint64 thresholdPercentage,
        uint64 taskStatisticalPeriod
    ) external returns (uint64 taskID);

    /// @dev challenge ,  this function enables a challenger to raise and resolve a challenge.
    /// @param sender The external address for calling this method.
    /// @param taskHash The data supplied by the contract, usually ABI-encoded.
    /// @param taskID The id of task.
    /// @param taskResponseHash The hash of task response.
    /// @param operatorAddress operator address.
    function challenge(
        address sender,
        bytes calldata taskHash,
        uint64 taskID,
        bytes calldata taskResponseHash,
        address  operatorAddress
    ) external returns (bool success);

    /// @dev Called by the avs manager service register an operator as the owner of a BLS public key.
    /// @param sender The external address for calling this method.
    /// @param name the name of public keys
    /// @param pubKey the public keys of the operator
    /// @param pubkeyRegistrationSignature the public keys of the operator
    /// @param pubkeyRegistrationMessageHash the public keys of the operator
    function registerBLSPublicKey(
        address sender,
        string calldata name,
        bytes calldata pubKey,
        bytes calldata pubkeyRegistrationSignature,
        bytes calldata pubkeyRegistrationMessageHash
    ) external returns (bool success);

    /// @dev operatorSubmitTask ,  this function enables a operator submit a task result.
    /// @param sender The external address for calling this method.
    /// @param taskID The id of task.
    /// @param taskResponse is the task response data..
    /// @param blsSignature is the operator bls sig info..
    /// @param taskContractAddress is contract address of task.
    /// @param phase The phase of the Two-Phase Commit protocol:
    ///             1 = Prepare phase (commit preparation)
    ///             2 = Commit phase (final commitment)
    function operatorSubmitTask(
        address sender,
        uint64 taskID,
        bytes calldata taskResponse,
        bytes calldata blsSignature,
        address taskContractAddress,
        uint8 phase
    ) external returns (bool success);

    /// QUERIES
    /// @dev Returns the pubkey and pubkey hash of an operator
    /// @param operator is the operator for whom the key is being registered
    function getRegisteredPubkey(address operator) external view returns (bytes memory pubkey);

    /// @dev Returns the operators of all opt-in in the current avs
    /// @param avsAddress avs address
    function getOptInOperators(address avsAddress) external view returns (string[] memory operators);

    /// @dev getAVSUSDValue is a function to retrieve the USD share of specified Avs.
    /// @param avsAddr The address of the avs
    /// @return amount The total USD share of specified operator and Avs.
    function getAVSUSDValue(address avsAddr) external view returns (uint256 amount);

    /// @dev getOperatorOptedUSDValue  is a function to retrieve the USD share of specified operator and Avs.
    /// @param avsAddr The address of the avs
    /// @param operatorAddr The address of the operator
    /// @return amount The total USD share of specified operator and Avs.
    function getOperatorOptedUSDValue(address avsAddr, address operatorAddr)
        external
        view
        returns (uint256 amount);

    /// @dev getAVSEpochIdentifier returns the epoch identifier for the given AVS.
    /// @param avsAddr The address of the avs
    function getAVSEpochIdentifier(address avsAddr) external view returns (string memory epochIdentifier);

    /// @dev getTaskInfo  is a function to query task info.
    /// @param taskAddr The address of the avs task
    /// @param taskID The id of task.
    /// @return info Array containing task information in the following order:
    /// [0] = startingEpochNumber
    /// [1] = taskResponsePeriod
    /// [2] = taskStatisticalPeriod
    function getTaskInfo(address taskAddr, uint64 taskID) external view returns (uint64[] memory info);

    /// @dev isOperator checks if the given address is registered as an operator on exocore.
    /// @param operatorAddr The address of the operator
    function isOperator(address operatorAddr) external view returns (bool);

    /// @dev getCurrentEpoch obtain the specified current epoch based on epochIdentifier.
    /// @param epochIdentifier  is a descriptive or unique identifier for the epoch
    function getCurrentEpoch(string memory epochIdentifier) external view returns (int64 currentEpoch);
}
