#![cfg_attr(target_arch = "wasm32", no_main)]

mod state;

use std::collections::BTreeSet;

use self::state::UniversalSolverState;
use linera_sdk::{base::WithContractAbi, Contract, ContractRuntime, DataBlobHash};
use linera_sdk::base::AccountOwner;
use linera_views::views::{RootView, View};
use universal_solver::{Operation, SolverFile, UniversalSolverAbi};
use async_graphql;

pub struct UniversalSolverContract {
    state: UniversalSolverState,
    runtime: ContractRuntime<Self>,
}

linera_sdk::contract!(UniversalSolverContract);

impl WithContractAbi for UniversalSolverContract {
    type Abi = UniversalSolverAbi;
}


impl Contract for UniversalSolverContract {
    type Message = ();
    type Parameters = ();
    type InstantiationArgument = ();
    async fn instantiate(&mut self, _argument: Self::InstantiationArgument) {
        self.runtime.application_parameters();
    }

    async fn execute_operation(&mut self, operation: Self::Operation) -> Self::Response {
        match operation {
            Operation::AddFile { owner, name, blob_hash } => {
                self.check_account_authentication(owner);
                self.add_file(owner, name, blob_hash).await;
            }
            Operation::AddPool { chain_name, pool_address } => {
                self.state.pool_list.insert(&chain_name, pool_address)
                    .expect("Failed to add pool");
            }
            Operation::RemovePool { chain_name } => {
                self.state.pool_list.remove(&chain_name)
                    .expect("Failed to remove pool");
            }
            Operation::UpdatePoolBalance { pool_address, balance } => {
                self.state.pool_balances.insert(&pool_address, balance)
                    .expect("Failed to update pool balance");
            }
            Operation::Swap { 
                from_token, 
                to_token, 
                destination_address,
                amount 
            } => {
                // Verify addresses exist in pool list
                let from_pool = self.state.pool_list.get(&from_token).await
                    .expect("Failed to get source pool")
                    .expect("Source pool not found");
                let to_pool = self.state.pool_list.get(&to_token).await
                    .expect("Failed to get target pool")
                    .expect("Target pool not found");

                // Track and validate the swap
                self.validate_and_track_swap(&from_pool, amount).await;

                // Calculate swap amount using service query
                let application_id = self.runtime.application_id();
                let request = async_graphql::Request::new(format!(
                    r#"query {{ calculateSwap(fromToken: "{from_token}", toToken: "{to_token}", amount: {amount}) }}"#
                ));

                let response = self.runtime.query_service(application_id, request);
                let async_graphql::Value::Object(data_object) = response.data else {
                    panic!("Unexpected response from `calculateSwap`: {response:?}");
                };

                let async_graphql::Value::Object(swap_result) = &data_object["calculateSwap"] else {
                    panic!("Missing `calculateSwap` result in response data: {data_object:?}");  
                };

                let async_graphql::Value::Number(to_amount_value) = &swap_result["toAmount"] else {
                    panic!("Missing `toAmount` in swap result data: {swap_result:?}");
                };

                let to_amount = to_amount_value.as_u64()
                    .expect("Invalid `toAmount` value in swap result");

                // Execute the swap
                self.execute_token_swap(
                    from_pool,
                    to_pool,
                    amount,
                    to_amount
                ).await;
            }
        }
    }

    async fn load(runtime: ContractRuntime<Self>) -> Self {
            let state = UniversalSolverState::load(runtime.root_view_storage_context())
                .await
                .expect("Failed to load state");
            UniversalSolverContract { state, runtime }
        }

        async fn store(mut self) {
            self.state.save().await.expect("Failed to save state");
        }

    async fn execute_message(&mut self, _message: Self::Message) {
        todo!()
    }
}

    impl UniversalSolverContract {
    fn check_account_authentication(&mut self, owner: AccountOwner) {
        match owner {
            AccountOwner::Application(id) => {
                assert_eq!(
                    self.runtime.authenticated_caller_id(),
                    Some(id),
                    "The requested transfer is not correctly authenticated."
                )
            }

            AccountOwner::User(address) => {
                assert_eq!(
                    self.runtime.authenticated_signer(),
                    Some(address),
                    "The requested transfer is not correctly authenticated."
                )
            }
        }
    }

    async fn add_file(&mut self, owner: AccountOwner, name: String, blob_hash: DataBlobHash) {
        self.runtime.assert_data_blob_exists(blob_hash);
        let file_app_id = SolverFile::create_token_id(
            &self.runtime.chain_id(),
            &self.runtime.application_id().forget_abi(),
            &name,
            &blob_hash,
        ).expect("Failed to serialize file metadata");

        self.add(SolverFile{
            solver_file_id: file_app_id,
            owner,
            name,
            blob_hash,
        }).await;
    }

    async fn add(&mut self, solver_file: SolverFile) {
        let file_id = solver_file.solver_file_id.clone();
        let owner = solver_file.owner;
        self.state.files.insert(&file_id, solver_file).expect("Error in file insert");
        if let Some(owned_files) = self
            .state.owned_files.get_mut(&owner).await.expect("Error in get_mut statement"){
            owned_files.insert(file_id);
        } else {
            let mut owned_files = BTreeSet::new();
            owned_files.insert(file_id);
            self.state.owned_files.insert(&owner, owned_files).expect("Error in insert statement");
        }
    }

    /// Validates and tracks a swap operation
    async fn validate_and_track_swap(&mut self, from_address: &str, amount: u64) {
        // Check balance
        let from_balance = self.state.pool_balances.get(from_address).await
            .expect("Failed to get source balance")
            .expect("Source balance not found");

        assert!(from_balance >= amount, "Insufficient balance");
    }

    /// Executes the token swap by updating balances
    async fn execute_token_swap(
        &mut self,
        from_address: String,
        to_address: String,
        from_amount: u64,
        to_amount: u64,
    ) {
        // Update source balance
        let mut from_balance = self.state.pool_balances.get(&from_address).await
            .expect("Failed to get source balance")
            .expect("Source balance not found");
        from_balance -= from_amount;
        self.state.pool_balances.insert(&from_address, from_balance)
            .expect("Failed to update source balance");

        // Update target balance
        let mut to_balance = self.state.pool_balances.get(&to_address).await
            .expect("Failed to get target balance")
            .unwrap_or(0);
        to_balance += to_amount;
        self.state.pool_balances.insert(&to_address, to_balance)
            .expect("Failed to update target balance");
    }
}