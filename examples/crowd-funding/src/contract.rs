// Copyright (c) Zefchain Labs, Inc.
// SPDX-License-Identifier: Apache-2.0

#![cfg_attr(target_arch = "wasm32", no_main)]

mod state;

use std::num::FpCategory::Zero;
use async_graphql::InputType;
use crowd_funding::{ChainAddresses, ChainPledges, CrowdApplication, CrowdFundingAbi, InstantiationArgument, Message, Operation, Status, TotalChainPledges};
use fungible::{Account, FungibleTokenAbi};
use linera_sdk::{
    base::{AccountOwner, Amount, ApplicationId, WithContractAbi},
    views::{RootView, View},
    Contract, ContractRuntime,
};
use state::{CrowdFundingState};

pub struct CrowdFundingContract {
    state: CrowdFundingState,
    runtime: ContractRuntime<Self>,
}

linera_sdk::contract!(CrowdFundingContract);

impl WithContractAbi for CrowdFundingContract {
    type Abi = CrowdFundingAbi;
}

impl Contract for CrowdFundingContract {
    type Message = Message;
    type InstantiationArgument = InstantiationArgument;
    type Parameters = ApplicationId<fungible::FungibleTokenAbi>;

    async fn load(runtime: ContractRuntime<Self>) -> Self {
        let state = CrowdFundingState::load(runtime.root_view_storage_context())
            .await
            .expect("Failed to load state");
        CrowdFundingContract { state, runtime }
    }

    async fn instantiate(&mut self, argument: InstantiationArgument) {
        // Validate that the application parameters were configured correctly.
        let _ = self.runtime.application_parameters();

        self.state.instantiation_argument.set(Some(argument));

        let deadline = self.instantiation_argument().deadline;
        assert!(
            deadline > self.runtime.system_time(),
            "Crowd-funding campaign cannot start after its deadline"
        );
    }

    async fn execute_operation(&mut self, operation: Operation) -> Self::Response {
        match operation {
            Operation::Pledge { owner, amount } => {
                if self.runtime.chain_id() == self.runtime.application_creator_chain_id() {
                    self.execute_pledge_with_account(owner, amount).await;
                } else {
                    self.execute_pledge_with_transfer(owner, amount);
                }
            }
            Operation::Collect { twitter_id} => self.collect_pledges(twitter_id).await,
            Operation::Cancel => self.cancel_campaign().await,
            Operation::AddChain { twitter_id, chain_name, chain_address} => {
               let mut app =  self.state.crowd_application.get(&twitter_id).await.expect("failed to get crowd app")
                   .expect("failed to get app");
                let mut found = false;
                for (index, total_pledge) in app.chain_addresses.clone().iter_mut().enumerate() {
                    if total_pledge.chain == chain_name {
                        total_pledge.address = chain_address.clone();
                        app.chain_addresses.insert(index, total_pledge.clone());
                        app.chain_addresses.remove(index);
                        found = true;
                        break;
                    }
                }

                if !found {
                    app.chain_addresses.push(ChainAddresses {
                        address: chain_address,
                        chain: chain_name,
                    });
                }
                self.state.crowd_application.insert(&twitter_id, app).expect("failed to insert chain address");
            }
            Operation::RemoveChain { chain_name } => {
                self.state.chain_addresses.remove(&chain_name)
                    .expect("Failed to remove chain address");
            }
            Operation::Fund {
                twitter_id,
                chain_name,
                deposit_address,
                amount,
            } => {

                let parse_amount =  amount.parse::<f64>().unwrap();

                // add amount, and record of deposit
                self.pledge_chain_amount(chain_name, deposit_address, parse_amount, twitter_id).await;

            }

            Operation::NewCrowdApp {
                args,
                profile_screenshot,
                twitter_id
            } => {
                self.runtime.assert_data_blob_exists(profile_screenshot);

                let app = CrowdApplication{
                    status: Default::default(),
                    profile_screenshot,
                    instantiation_argument: args,
                    chain_addresses: Default::default(),
                    total_chain_pledges: Default::default(),
                    individual_pledges: Default::default(),
                };
                self.state.crowd_application.insert(&twitter_id, app).unwrap()
            }
        }
    }

    async fn execute_message(&mut self, message: Message) {
        match message {
            Message::PledgeWithAccount { owner, amount } => {
                assert_eq!(
                    self.runtime.chain_id(),
                    self.runtime.application_creator_chain_id(),
                    "Action can only be executed on the chain that created the crowd-funding \
                    campaign"
                );
                self.execute_pledge_with_account(owner, amount).await;
            }
        }
    }

    async fn store(mut self) {
        self.state.save().await.expect("Failed to save state");
    }
}

impl CrowdFundingContract {
    async fn get_crowd_app(&mut self, twitter_id: String) -> CrowdApplication {
        let mut app =  self.state.crowd_application.get(&twitter_id).await.expect("failed to get crowd app")
            .expect("failed to get app");
        app
    }

    async fn pledge_chain_amount(&mut self, chain_name: String, deposit_address: String, amount: f64, twitter_id: String) {
        let key = format!("{}_{}", deposit_address, chain_name);

        let mut app = self.get_crowd_app(twitter_id.clone()).await;

        app.individual_pledges.push(ChainPledges {
            deposit_address: key,
            amount: amount.to_string(),
        });

        self.state.crowd_application.insert(&twitter_id, app).expect("failed to insert app");
        self.execute_total_chain_pledges( twitter_id, chain_name, amount).await;

    }

    async fn execute_total_chain_pledges(&mut self, twitter_id: String, chain_name: String, amount: f64) {
        let mut app = self.get_crowd_app(twitter_id.clone()).await;

        let mut found = false;
        for (index, total_pledge) in app.total_chain_pledges.clone().iter_mut().enumerate() {
            if total_pledge.chain == chain_name {
                let curr_amt  = total_pledge.amount.is_empty().then(|| "0".to_string()).unwrap_or(total_pledge.clone().amount);
                let mut curr_amt_f64: f64 = curr_amt.parse().unwrap();
                curr_amt_f64 += amount;
                total_pledge.amount = curr_amt_f64.to_string();
                app.total_chain_pledges.remove(index);
                app.total_chain_pledges.insert(index, total_pledge.clone());
                found = true;
                break;
            }
        }

        if !found {
            app.total_chain_pledges.push(TotalChainPledges {
                chain: chain_name,
                amount: amount.to_string(),
            });
        }

        self.state.crowd_application.insert(&twitter_id, app).expect("failed to insert app");
    }

    fn fungible_id(&mut self) -> ApplicationId<FungibleTokenAbi> {
        // TODO(#723): We should be able to pull the fungible ID from the
        // `required_application_ids` of the application description.
        self.runtime.application_parameters()
    }

    /// Adds a pledge from a local account to the remote campaign chain.
    fn execute_pledge_with_transfer(&mut self, owner: AccountOwner, amount: Amount) {
        assert!(amount > Amount::ZERO, "Pledge is empty");
        // The campaign chain.
        let chain_id = self.runtime.application_creator_chain_id();
        // First, move the funds to the campaign chain (under the same owner).
        // TODO(#589): Simplify this when the messaging system guarantees atomic delivery
        // of all messages created in the same operation/message.
        let target_account = Account { chain_id, owner };
        let call = fungible::Operation::Transfer {
            owner,
            amount,
            target_account,
        };
        let fungible_id = self.fungible_id();
        self.runtime
            .call_application(/* authenticated by owner */ true, fungible_id, &call);
        // Second, schedule the attribution of the funds to the (remote) campaign.
        self.runtime
            .prepare_message(Message::PledgeWithAccount { owner, amount })
            .with_authentication()
            .send_to(chain_id);
    }

    /// Adds a pledge from a local account to the campaign chain.
    async fn execute_pledge_with_account(&mut self, owner: AccountOwner, amount: Amount) {
        assert!(amount > Amount::ZERO, "Pledge is empty");
        self.receive_from_account(owner, amount);
        self.finish_pledge(owner, amount).await
    }

    /// Marks a pledge in the application state, so that it can be returned if the campaign is
    /// cancelled.
    async fn finish_pledge(&mut self, source: AccountOwner, amount: Amount) {
        match self.state.status.get() {
            Status::Active => self
                .state
                .pledges
                .get_mut_or_default(&source)
                .await
                .expect("view access should not fail")
                .saturating_add_assign(amount),
            Status::Complete => self.send_to(amount, self.instantiation_argument().owner),
            Status::Cancelled => panic!("Crowd-funding campaign has been cancelled"),
        }
    }


    async fn total_in_usd(&mut self, twitter_id: String) -> f64 {
        let application_id = self.runtime.application_id();
        let mut app = self.get_crowd_app(twitter_id.clone()).await;
        let mut total: f64 = 0.0;
        for total_pledge in app.total_chain_pledges.iter_mut() {
            let chain = total_pledge.chain.clone();
            let amount = total_pledge.amount.to_string();

            let request = async_graphql::Request::new(format!(
                r#"query {{ fetchTokenPrice(token: "{chain}") {{ price }} }}"#
            ));

            let response = self.runtime.query_service(application_id, request);
            let async_graphql::Value::Object(data_object) = response.data else {
                panic!("Unexpected response from `fetchTokenPrice`: {response:?}");
            };

            let result = match data_object.get("fetchTokenPrice") {
                Some(async_graphql::Value::Object(result)) => result,
                _ => panic!("Missing or invalid fetchTokenPrice result in response data: {data_object:?}")
            };

            let price = match result.get("price") {
                Some(async_graphql::Value::Number(n)) => n.as_f64().unwrap(),
                _ => panic!("Invalid price in  result: {result:?}")
            };

            let curr_amt_f64: f64 = amount.parse().unwrap();
            total += price * curr_amt_f64;
        };

        total
    }

    /// Collects all pledges and completes the campaign if the target has been reached.
    async fn collect_pledges(&mut self, twitter_id: String) {
        let total = self.total_in_usd(twitter_id).await;
        match self.state.status.get() {
            Status::Active => {
                assert!(
                    total >= self.instantiation_argument().target.parse().unwrap(),
                    "Crowd-funding campaign has not reached its target yet"
                );
            }
            Status::Complete => (),
            Status::Cancelled => panic!("Crowd-funding campaign has been cancelled"),
        }
        self.state.status.set(Status::Complete);
    }

    /// Cancels the campaign if the deadline has passed, refunding all pledges.
    async fn cancel_campaign(&mut self) {
        assert!(
            !self.state.status.get().is_complete(),
            "Crowd-funding campaign has already been completed"
        );

        // TODO(#728): Remove this.
        #[cfg(not(test))]
        assert!(
            self.runtime.system_time() >= self.instantiation_argument().deadline,
            "Crowd-funding campaign has not reached its deadline yet"
        );

        let mut pledges = Vec::new();
        self.state
            .pledges
            .for_each_index_value(|pledger, amount| {
                let amount = amount.into_owned();
                pledges.push((pledger, amount));
                Ok(())
            })
            .await
            .expect("view iteration should not fail");
        for (pledger, amount) in pledges {
            self.send_to(amount, pledger);
        }

        let balance = self.balance();
        self.send_to(balance, self.instantiation_argument().owner);
        self.state.status.set(Status::Cancelled);
    }

    /// Queries the token application to determine the total amount of tokens in custody.
    fn balance(&mut self) -> Amount {
        let owner = AccountOwner::Application(self.runtime.application_id().forget_abi());
        let fungible_id = self.fungible_id();
        let response = self.runtime.call_application(
            true,
            fungible_id,
            &fungible::Operation::Balance { owner },
        );
        match response {
            fungible::FungibleResponse::Balance(balance) => balance,
            response => panic!("Unexpected response from fungible token application: {response:?}"),
        }
    }

    /// Transfers `amount` tokens from the funds in custody to the `owner`'s account.
    fn send_to(&mut self, amount: Amount, owner: AccountOwner) {
        let target_account = Account {
            chain_id: self.runtime.chain_id(),
            owner,
        };
        let transfer = fungible::Operation::Transfer {
            owner: AccountOwner::Application(self.runtime.application_id().forget_abi()),
            amount,
            target_account,
        };
        let fungible_id = self.fungible_id();
        self.runtime.call_application(true, fungible_id, &transfer);
    }

    /// Calls into the Fungible Token application to receive tokens from the given account.
    fn receive_from_account(&mut self, owner: AccountOwner, amount: Amount) {
        let target_account = Account {
            chain_id: self.runtime.chain_id(),
            owner: AccountOwner::Application(self.runtime.application_id().forget_abi()),
        };
        let transfer = fungible::Operation::Transfer {
            owner,
            amount,
            target_account,
        };
        let fungible_id = self.fungible_id();
        self.runtime.call_application(true, fungible_id, &transfer);
    }

    pub fn instantiation_argument(&self) -> &InstantiationArgument {
        self.state
            .instantiation_argument
            .get()
            .as_ref()
            .expect("Application is not running on the host chain or was not instantiated yet")
    }
}
