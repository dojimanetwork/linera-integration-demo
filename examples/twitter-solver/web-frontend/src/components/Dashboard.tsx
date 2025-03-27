import React, {useEffect, useState} from 'react';
import { Link, Box, Typography, Button, Paper, Avatar } from '@mui/material';
import { useAuth } from '../contexts/AuthContext';
import api, {Tweet} from "../services/api";
import CameraAltIcon from "@mui/icons-material/CameraAlt";
import {useNavigate} from 'react-router-dom';

export const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [tweet, setTweet] = useState<Tweet | null>(null);
  const [screenshotUrl, setScreenshotUrl] = useState<string | null>(null);
  const [isTakingScreenshot, setIsTakingScreenshot] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    const handleLatestTweet = async () => {
      try {
        // Fetch the latest tweet
        const tweetResponse = await api.getLatestTweet();
        if (tweetResponse.status === 'success' && tweetResponse.tweet) {
          setTweet(tweetResponse.tweet);
        }
      }catch (err) {
        setError(err instanceof Error ? err.message : 'An error occurred');
      }
    }
    handleLatestTweet()
  },[])

  const handleTakeScreenshot = async () => {
    if (!tweet) return;

    setIsTakingScreenshot(true);
    try {
      const tweetUrl = `https://twitter.com/user/status/${tweet.id}`;
      const response = await api.takeScreenshot(tweetUrl);
      if (response.status === 'success') {
        setScreenshotUrl(response.screenshot_url);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to take screenshot');
    } finally {
      setIsTakingScreenshot(false);
    }
  };


  if (error) {
    return (
        <Box sx={{ p: 3, textAlign: 'center' }}>
          <Typography color="error" gutterBottom>
            Error: {error}
          </Typography>
          <Button variant="contained" onClick={() => navigate('/')}>
            Return to Home
          </Button>
        </Box>
    );
  }

  return (
    <Box display="flex" flexDirection="column" alignItems="center" minHeight="100vh" p={3}>
      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%' }}>
        <Box display="flex" alignItems="center" mb={3}>
          <Avatar sx={{ bgcolor: '#1DA1F2', mr: 2 }}>
            {user?.username[0].toUpperCase()}
          </Avatar>
          <Box>
            <Typography variant="h6">{user?.name}</Typography>
            <Typography color="text.secondary">@{user?.username}</Typography>
          </Box>
        </Box>


        {tweet && (
            <Paper sx={{ p: 3, mt: 3 }}>
              <Typography variant="h6" gutterBottom>
                Latest Tweet
              </Typography>
              <Typography variant="body1" paragraph>
                {tweet.text}
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block" gutterBottom>
                Posted on {new Date(tweet.created_at).toLocaleString()}
              </Typography>
              <Box sx={{ mt: 2, display: 'flex', gap: 2 }}>
                <Button
                    variant="outlined"
                    startIcon={<CameraAltIcon />}
                    onClick={handleTakeScreenshot}
                    disabled={isTakingScreenshot}
                >
                  {isTakingScreenshot ? 'Taking Screenshot...' : 'Take Screenshot'}
                </Button>
                <Link
                    href={`https://twitter.com/user/status/${tweet.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                >
                  View on Twitter ${tweet.id}
                </Link>
              </Box>
              {screenshotUrl && (
                  <Box sx={{ mt: 2 }}>
                    <img
                        src={`http://localhost:8080/${screenshotUrl}`}
                        alt="Tweet Screenshot"
                        style={{ maxWidth: '100%', height: 'auto' }}
                    />

                  </Box>
              )}
            </Paper>
        )}
      </Paper>
    </Box>
  );
}; 
