import axios from 'axios';

const API_BASE_URL = 'http://localhost:8080';
const REDIRECT_URI = 'http://localhost:5173/twitter/callback';

export interface TwitterAuthResponse {
  auth_url: string;
  code_verifier: string;
}

export interface TwitterCallbackResponse {
  status: string;
  user?: {
    username: string;
    name: string;
  };
  error?: string;
}

export interface TweetRequest {
  content: string;
  author: string;
}

export interface TweetEntities {
  mentions: any[] | null;
  hashtags: any[] | null;
  annotations: any[] | null;
}

export interface Tweet {
  id: string;
  text: string;
  created_at: string;
  author_id: string;
  display_text_range: [number, number];
  edit_history_tweet_ids: string[];
  entities: TweetEntities;
}

export interface LatestTweetResponse {
  status: string;
  tweet: Tweet;
}

const api = {
  // Get Twitter auth URL
  getTwitterAuthUrl: async (): Promise<TwitterAuthResponse> => {
    const response = await axios.get(`${API_BASE_URL}/twitter/auth`, {
      params: { redirect_uri: REDIRECT_URI }
    });
    return response.data;
  },

  // Handle Twitter callback
  handleTwitterCallback: async (code: string, state: string, codeVerifier: string): Promise<TwitterCallbackResponse> => {
    try {
      const response = await fetch(`${API_BASE_URL}/twitter/callback?code=${code}&state=${state}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ code_verifier: codeVerifier }),
      });

      if (!response.ok) {
        throw new Error('Failed to handle Twitter callback');
      }

      return await response.json();
    } catch (error) {
      console.error('Error handling Twitter callback:', error);
      throw error;
    }
  },

  // Post a tweet
  postTweet: async (tweet: TweetRequest): Promise<void> => {
    await axios.post(`${API_BASE_URL}/post_tweet`, tweet);
  },

  // Get all tweets
  getTweets: async () => {
    const response = await axios.get(`${API_BASE_URL}/get_tweets`);
    return response.data;
  },

  // Get latest tweet
  getLatestTweet: async (): Promise<LatestTweetResponse> => {
    try {
      const response = await fetch(`${API_BASE_URL}/latest_tweet`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error('Failed to fetch latest tweet');
      }

      return await response.json();
    } catch (error) {
      console.error('Error fetching latest tweet:', error);
      throw error;
    }
  },

  // Take screenshot of a tweet
  takeScreenshot: async (tweetUrl: string): Promise<{ status: string; screenshot_url: string }> => {
    try {
      const response = await fetch(`${API_BASE_URL}/take_screenshot`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ tweet_url: tweetUrl }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.detail || 'Failed to take screenshot');
      }

      const data = await response.json();
      return data;
    } catch (error) {
      console.error('Error taking screenshot:', error);
      throw error;
    }
  }
}

export default api; 