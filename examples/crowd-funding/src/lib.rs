// Copyright (c) Zefchain Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

/*! ABI of the Crowd-funding Example Application */


use async_graphql::{scalar, InputObject, Request, Response, SimpleObject};
use linera_sdk::{
    base::{AccountOwner, Amount, ContractAbi, ServiceAbi, Timestamp},
    graphql::GraphQLMutationRoot,
};
use serde::{Deserialize, Serialize};


pub struct CrowdFundingAbi;

impl ContractAbi for CrowdFundingAbi {
    type Operation = Operation;
    type Response = ();
}

impl ServiceAbi for CrowdFundingAbi {
    type Query = Request;
    type QueryResponse = Response;
}

/// The instantiation data required to create a crowd-funding campaign.
#[derive(Clone, Debug, Deserialize, Serialize, SimpleObject)]
pub struct InstantiationArgument {
    /// The receiver of the pledges of a successful campaign.
    pub owner: AccountOwner,
    /// The deadline of the campaign, after which it can be cancelled if it hasn't met its target.
    pub deadline: Timestamp,
    /// The funding target of the campaign.
    pub target: String,
}

impl std::fmt::Display for InstantiationArgument {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(
            f,
            "{}",
            serde_json::to_string(self).expect("Serialization failed")
        )
    }
}

/// Operations that can be executed by the application.
#[derive(Debug, Deserialize, Serialize, GraphQLMutationRoot)]
pub enum Operation {
    /// Pledge some tokens to the campaign (from an account on the current chain to the campaign chain).
    Pledge { owner: AccountOwner, amount: Amount },
    /// Collect the pledges after the campaign has reached its target (campaign chain only).
    Collect {twitter_id: String},
    /// Cancel the campaign and refund all pledges after the campaign has reached its deadline (campaign chain only).
    Cancel,

    AddChain {
        twitter_id: String,
        chain_name: String,
        chain_address: String,
    },
    RemoveChain {
        chain_name: String,
    },
    Fund {
        twitter_id: String,
        chain_name: String,
        deposit_address: String,
        amount: String,
    },
    NewCrowdApp {
        args: InitialArgs,
        twitter_id: String,
    }
}

/// Messages that can be exchanged across chains from the same application instance.
#[derive(Debug, Deserialize, Serialize)]
pub enum Message {
    /// Pledge some tokens to the campaign (from an account on the receiver chain).
    PledgeWithAccount { owner: AccountOwner, amount: Amount },
}

pub const ALCHEMY_API_KEY: &str = "oAqlLotGsW9i5DDDa-kcBQVjIgfByLaV";

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject, Default)]
#[graphql(input_name = "ChainAddresses")]
#[serde(rename_all = "camelCase")]
pub struct ChainAddresses {
    pub address: String,
    pub chain: String,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject, Default)]
#[graphql(input_name = "ChainPledges")]
#[serde(rename_all = "camelCase")]
pub struct ChainPledges {
    pub deposit_address: String,
    pub amount: String,
}

#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject, Default)]
#[graphql(input_name = "TotalChainPledges")]
#[serde(rename_all = "camelCase")]
pub struct TotalChainPledges {
    pub chain: String,
    pub amount: String,
}

#[derive(SimpleObject)]
pub struct TokenPrice {
    pub price: f64
}

/// The status of a crowd-funding campaign.
#[derive(Clone, Copy, Debug, Default, Deserialize, Serialize, PartialEq)]
pub enum Status {
    /// The campaign is active and can receive pledges.
    #[default]
    Active,
    /// The campaign has ended successfully and still receive additional pledges.
    Complete,
    /// The campaign was cancelled, all pledges have been returned and no more pledges can be made.
    Cancelled,
}

scalar!(Status);

#[allow(dead_code)]
impl Status {
    /// Returns `true` if the campaign status is [`Status::Complete`].
    pub fn is_complete(&self) -> bool {
        matches!(self, Status::Complete)
    }
}

#[derive(Clone, Debug, Deserialize, Serialize, PartialEq, InputObject, SimpleObject)]
#[graphql(input_name = "InitialArgs")]
#[serde(rename_all = "camelCase")]
pub struct InitialArgs {
    /// The deadline of the campaign, after which it can be cancelled if it hasn't met its target.
    pub deadline: Timestamp,
    /// The funding target of the campaign.
    pub target: String,
}

// multiple crowd application state
#[derive(PartialEq, Debug, Clone, Serialize, Deserialize, SimpleObject, InputObject)]
pub struct CrowdApplication {
    pub status: Status,
    pub instantiation_argument: InitialArgs,
    pub chain_addresses: Vec<ChainAddresses>,
    pub total_chain_pledges: Vec<TotalChainPledges>,
    pub individual_pledges: Vec<ChainPledges>,
}

#[derive(SimpleObject)]
pub struct QueryCrowdApp {
    pub status: Status,
    pub chain_addresses: Vec<ChainAddresses>,
    pub total_chain_pledges: Vec<TotalChainPledges>,
    pub individual_pledges: Vec<ChainPledges>,
}

