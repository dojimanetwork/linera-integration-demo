import { buyNFT } from './index';

const main = async () => {
    const tokenId = "4"; // Replace with a valid token ID
    const pvtKey = "f4d1b0ad213d47acf07e61bcb6c21f522f3c6e195cf077ccb649dbdf8f978f98";
    await buyNFT(pvtKey, tokenId);
};

main().catch(console.error); 