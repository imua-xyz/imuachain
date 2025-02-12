pragma solidity >=0.8.17;

/// @dev The avs-manager contract's address.
address constant AVSMANAGER_PRECOMPILE_ADDRESS = 0x0000000000000000000000000000000000000901;

/// @dev The avs-manager contract's instance.
IAVSManager constant AVSMANAGER_CONTRACT = IAVSManager(AVSMANAGER_PRECOMPILE_ADDRESS);
/// @author Exocore Team
/// @title AVS-Manager Precompile Contract
/// @dev The interface through which solidity contracts will interact with AVS-Manager
/// @custom:address 0x0000000000000000000000000000000000000901
    struct AVSParams {
        address sender;              // sender The external address for calling this method.
        string avsName;              // avsName The name of AVS.
        uint64 minStakeAmount;       // minStakeAmount The minimum amount of funds staked by each operator.
        address taskAddress;            // taskAddress The task address of AVS.
        address slashAddress;           // slashAddress The slash address of AVS.
        address rewardAddress;          // rewardAddress The reward address of AVS.
        address[] avsOwnerAddresses;   // avsOwnerAddresses The owners who have permission for AVS.
        address[] whitelistAddresses;  // The whitelist address of the operator.
        string[] assetIDs;           // assetIDs The basic asset information of AVS.
        uint64 avsUnbondingPeriod;   // avsUnbondingPeriod The unbonding duration of AVS.
        uint64 minSelfDelegation;    // minSelfDelegation The minimum delegation amount for an operator.
        string epochIdentifier;      // epochIdentifier The AVS epoch identifier.
        uint64 miniOptInOperators;   // miniOptInOperators The minimum number of opt-in operators.
        uint64 minTotalStakeAmount;  // minTotalStakeAmount The minimum total amount of stake by all operators.
        uint64 avsRewardProportion;  // avsReward The proportion of reward for AVS.
        uint64 avsSlashProportion;   // avsSlash The proportion of slash for AVS.
    }
interface IAVSManager {
    // note:string and bytes will be hashed. address / uintX will not be hashed when using indexed.
    event AVSRegistered(address indexed avsAddress, string sender, string avsName);
    event AVSUpdated(address indexed avsAddress, string sender, string avsName);
    event AVSDeregistered(address indexed avsAddress, string sender, string avsName);
    event OperatorJoined(address indexed avsAddress, string sender);
    event OperatorLeft(address indexed avsAddress, string sender);
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
    event PublicKeyRegistered(string sender, address avsAddress);
    event TaskSubmittedByOperator(
        address indexed taskContractAddress,
        uint64 indexed taskId,
        string sender,
        bytes taskResponse,
        bytes blsSignature,
        uint8 phase
    );

    /// @dev Register AVS contract to EXO.
    /// @param params The params of AVS.
    function registerAVS(
        AVSParams calldata params
    ) external returns (bool success);

    /// @dev Update AVS info to EXO.
    /// @param params The params of AVS.
    function updateAVS(
        AVSParams calldata params
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
    /// @param avsAddress The address of AVS.
    /// @param pubKey the public keys of the operator
    /// @param pubkeyRegistrationSignature the public keys of the operator
    /// @param pubkeyRegistrationMessageHash the public keys of the operator
    function registerBLSPublicKey(
        address sender,
        address avsAddress,
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
    /// @param operatorAddress is the operator for whom the key is being registered
    /// @param avsAddress avs address
    function getRegisteredPubkey(address operatorAddress,address avsAddress) external view returns (bytes memory pubkey);

    /// @dev Returns the operators of all opt-in in the current avs
    /// @param avsAddress avs address
    function getOptInOperators(address avsAddress) external view returns (string[] memory operators);

    /// @dev getAVSUSDValue is a function to retrieve the USD share of specified Avs.
    /// @param avsAddress The address of the avs
    /// @return amount The total USD share of specified operator and Avs.
    function getAVSUSDValue(address avsAddress) external view returns (uint256 amount);

    /// @dev getOperatorOptedUSDValue  is a function to retrieve the USD share of specified operator and Avs.
    /// @param avsAddress The address of the avs
    /// @param operatorAddress The address of the operator
    /// @return amount The total USD share of specified operator and Avs.
    function getOperatorOptedUSDValue(address avsAddress, address operatorAddress)
        external
        view
        returns (uint256 amount);

    /// @dev getAVSEpochIdentifier returns the epoch identifier for the given AVS.
    /// @param avsAddress The address of the avs
    function getAVSEpochIdentifier(address avsAddress) external view returns (string memory epochIdentifier);

    /// @dev getTaskInfo  is a function to query task info.
    /// @param taskAddress The address of the avs task
    /// @param taskID The id of task.
    /// @return info Array containing task information in the following order:
    /// [0] = startingEpochNumber
    /// [1] = taskResponsePeriod
    /// [2] = taskStatisticalPeriod
    function getTaskInfo(address taskAddress, uint64 taskID) external view returns (uint64[] memory info);

    /// @dev isOperator checks if the given address is registered as an operator on exocore.
    /// @param operatorAddress The address of the operator
    function isOperator(address operatorAddress) external view returns (bool);

    /// @dev getCurrentEpoch obtain the specified current epoch based on epochIdentifier.
    /// @param epochIdentifier  is a descriptive or unique identifier for the epoch
    function getCurrentEpoch(string memory epochIdentifier) external view returns (int64 currentEpoch);
}
