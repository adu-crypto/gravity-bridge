use crate::types::EthClient;
use ethers::middleware::gas_oracle::Etherscan;
use ethers::prelude::gas_oracle::GasOracle;
use ethers::prelude::*;
use ethers::types::Address as EthAddress;
use gravity_abi::gravity::*;
use gravity_utils::error::GravityError;
use gravity_utils::ethereum::{downcast_to_u64, vec_u8_to_fixed_32};
use std::result::Result;

/// Gets the latest validator set nonce
pub async fn get_valset_nonce(
    gravity_contract_address: EthAddress,
    eth_client: EthClient,
) -> Result<u64, GravityError> {
    let contract_call = Gravity::new(gravity_contract_address, eth_client.clone())
        .state_last_valset_nonce()
        .from(eth_client.address())
        .value(U256::zero());
    let gas_estimate = contract_call.estimate_gas().await?;
    let contract_call = contract_call
        .gas(gas_estimate)
        .gas_price(get_gas_price(eth_client.clone()).await?);

    let valset_nonce = contract_call.call().await?;

    // TODO (bolten): do we actually want to halt the bridge as the original comment implies?
    // the go represents all nonces as u64, there's no
    // reason they should ever overflow without a user
    // submitting millions or tens of millions of dollars
    // worth of transactions. But we properly check and
    // handle that case here.
    Ok(downcast_to_u64(valset_nonce).expect("Valset nonce overflow! Bridge Halt!"))
}

/// Gets the latest transaction batch nonce
pub async fn get_tx_batch_nonce(
    gravity_contract_address: EthAddress,
    erc20_contract_address: EthAddress,
    eth_client: EthClient,
) -> Result<u64, GravityError> {
    let contract_call = Gravity::new(gravity_contract_address, eth_client.clone())
        .last_batch_nonce(erc20_contract_address)
        .from(eth_client.address())
        .value(U256::zero());
    let gas_estimate = contract_call.estimate_gas().await?;
    let contract_call = contract_call
        .gas(gas_estimate)
        .gas_price(get_gas_price(eth_client.clone()).await?);

    let tx_batch_nonce = contract_call.call().await?;

    // TODO (bolten): do we actually want to halt the bridge as the original comment implies?
    // the go represents all nonces as u64, there's no
    // reason they should ever overflow without a user
    // submitting millions or tens of millions of dollars
    // worth of transactions. But we properly check and
    // handle that case here.
    Ok(downcast_to_u64(tx_batch_nonce).expect("TxBatch nonce overflow! Bridge Halt!"))
}

/// Gets the latest transaction batch nonce
pub async fn get_logic_call_nonce(
    gravity_contract_address: EthAddress,
    invalidation_id: Vec<u8>,
    eth_client: EthClient,
) -> Result<u64, GravityError> {
    let invalidation_id = vec_u8_to_fixed_32(invalidation_id)?;

    let contract_call = Gravity::new(gravity_contract_address, eth_client.clone())
        .last_logic_call_nonce(invalidation_id)
        .from(eth_client.address())
        .value(U256::zero());
    let gas_estimate = contract_call.estimate_gas().await?;
    let contract_call = contract_call
        .gas(gas_estimate)
        .gas_price(get_gas_price(eth_client.clone()).await?);

    let logic_call_nonce = contract_call.call().await?;

    // TODO (bolten): do we actually want to halt the bridge as the original comment implies?
    // the go represents all nonces as u64, there's no
    // reason they should ever overflow without a user
    // submitting millions or tens of millions of dollars
    // worth of transactions. But we properly check and
    // handle that case here.
    Ok(downcast_to_u64(logic_call_nonce).expect("LogicCall nonce overflow! Bridge Halt!"))
}

/// Gets the latest transaction batch nonce
pub async fn get_event_nonce(
    gravity_contract_address: EthAddress,
    eth_client: EthClient,
) -> Result<u64, GravityError> {
    let contract_call = Gravity::new(gravity_contract_address, eth_client.clone())
        .state_last_event_nonce()
        .from(eth_client.address())
        .value(U256::zero());
    let gas_estimate = contract_call.estimate_gas().await?;
    let contract_call = contract_call
        .gas(gas_estimate)
        .gas_price(get_gas_price(eth_client.clone()).await?);

    let event_nonce = contract_call.call().await?;

    // TODO (bolten): do we actually want to halt the bridge as the original comment implies?
    // the go represents all nonces as u64, there's no
    // reason they should ever overflow without a user
    // submitting millions or tens of millions of dollars
    // worth of transactions. But we properly check and
    // handle that case here.
    Ok(downcast_to_u64(event_nonce).expect("EventNonce nonce overflow! Bridge Halt!"))
}

/// Gets the gravityID
pub async fn get_gravity_id(
    gravity_contract_address: EthAddress,
    eth_client: EthClient,
) -> Result<String, GravityError> {
    let contract_call = Gravity::new(gravity_contract_address, eth_client.clone())
        .state_gravity_id()
        .from(eth_client.address())
        .value(U256::zero());
    let gas_estimate = contract_call.estimate_gas().await?;
    let contract_call = contract_call
        .gas(gas_estimate)
        .gas_price(get_gas_price(eth_client.clone()).await?);

    let gravity_id = contract_call.call().await?;
    let id_as_string = String::from_utf8(gravity_id.to_vec());

    match id_as_string {
        Ok(id) => Ok(id),
        Err(err) => Err(GravityError::GravityContractError(format!(
            "Received invalid utf8 when getting gravity id {:?}: {}",
            &gravity_id, err
        ))),
    }
}

/// If ETHERSCAN_API_KEY env var is set, we'll call out to Etherscan for a gas estimate.
/// Otherwise, just call eth_gasPrice.
pub async fn get_gas_price(eth_client: EthClient) -> Result<U256, GravityError> {
    if let Ok(_) = std::env::var("ETHERSCAN_API_KEY") {
        let chain = get_chain(eth_client.clone()).await?;
        let etherscan_client = Client::new_from_env(chain)?;
        let etherscan_oracle = Etherscan::new(etherscan_client);
        return Ok(etherscan_oracle.fetch().await?);
    }

    Ok(eth_client.get_gas_price().await?)
}

pub async fn get_chain(eth_client: EthClient) -> Result<Chain, GravityError> {
    let chain_id_result = eth_client.get_chainid().await?;
    let chain_id = downcast_to_u64(chain_id_result);

    if chain_id.is_none() {
        return Err(GravityError::EthereumBadDataError(format!(
            "Chain ID is larger than u64 max: {}",
            chain_id_result
        )));
    }

    // We're only currently looking for ETHERSCAN_API_KEY, so only support
    // Ethereum networks. Returning mainnet as a default in absence of a better
    // option. Strangely there is no function in ethers to convert from a chain
    // ID to a Chain enum value.
    Ok(match chain_id.unwrap() {
        1 => Chain::Mainnet,
        3 => Chain::Ropsten,
        4 => Chain::Rinkeby,
        5 => Chain::Goerli,
        42 => Chain::Kovan,
        _ => Chain::Mainnet,
    })
}

/// Just a helper struct to represent the cost of actions on Ethereum
#[derive(Debug, Default, Clone)]
pub struct GasCost {
    pub gas: U256,
    pub gas_price: U256,
}

impl GasCost {
    pub fn get_total(&self) -> U256 {
        self.gas * self.gas_price
    }
}
