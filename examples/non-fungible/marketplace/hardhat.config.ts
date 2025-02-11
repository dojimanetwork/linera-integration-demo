import { HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const config: HardhatUserConfig = {
  solidity: {
    version: "0.8.19", // your solidity version
    settings: {
      optimizer: {
        enabled: true,
        runs: 200,
      },
      viaIR: true, // Add this line
    },
  },
  paths: {
    artifacts: "./artifacts",
    cache: "./cache",
    sources: "./contracts",
    tests: "./test",
  },
  networks: {
    eth_dojima: {
      url: "https://eth-test.dojima.network",
      chainId: 1337,
      accounts: [
          // "ae1d058b9c5713848e7ac4c1901fac9a737729a34c10c997991c861dd7705ac0"
          "f2ec543791fe37da04c7a51795041750cf071cef41fb5c5e05f8c658fea6a8c6"
      ]
    },
  }
};

export default config;
