// Copyright (c) Zefchain Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

use async_graphql::scalar;
use crowd_funding::{CrowdApplication, InstantiationArgument, Status};
use linera_sdk::{
    base::{AccountOwner, Amount},
    views::{linera_views, MapView, RegisterView, RootView, ViewStorageContext},
};
use serde::{Deserialize, Serialize};



/// The crowd-funding campaign's state.
#[derive(RootView, async_graphql::SimpleObject)]
#[view(context = "ViewStorageContext")]
pub struct CrowdFundingState {
    /// The status of the campaign.
    pub status: RegisterView<Status>,
    /// The map of pledges that will be collected if the campaign succeeds.
    pub pledges: MapView<AccountOwner, Amount>,
    /// The instantiation data that determine the details the campaign.
    pub instantiation_argument: RegisterView<Option<InstantiationArgument>>,

    pub chain_addresses: MapView<String,String>,

    pub total_chain_pledges: MapView<String, String>,

    pub individual_pledges: MapView<String, String>,

    pub crowd_application: MapView<String,CrowdApplication>, // <Twitter_Id, CrowdApplication State>
}
