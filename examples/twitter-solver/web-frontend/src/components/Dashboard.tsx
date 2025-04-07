import React, {useEffect, useState} from 'react';
import { Link, Box, Typography, Button, Paper, Avatar, TextField, FormControlLabel, Switch } from '@mui/material';
import { useAuth } from '../contexts/AuthContext';
import api, {Tweet, WebhookNotification} from "../services/api";
import CameraAltIcon from "@mui/icons-material/CameraAlt";
import {useNavigate} from 'react-router-dom';
import UserProfile from './UserProfile';
import WebhookReceiver from './WebhookReceiver';
import { addWebhook } from '../services/webhookHandler';

export const Dashboard: React.FC = () => {
  const { user } = useAuth();
  const [tweet, setTweet] = useState<Tweet | null>(null);
  const [screenshotUrl, setScreenshotUrl] = useState<string | null>(null);
  const [profileScreenshotUrl, setProfileScreenshotUrl] = useState<string | null>(null);
  const [isTakingScreenshot, setIsTakingScreenshot] = useState(false);
  const [isTakingProfileScreenshot, setIsTakingProfileScreenshot] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [txHash, setTxHash] = useState('');
  const [chain, setChain] = useState('solana');
  const [webhookUrl, setWebhookUrl] = useState('');
  const [useWebhook, setUseWebhook] = useState(false);
  const [webhookStatus, setWebhookStatus] = useState<WebhookNotification | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedWebhookOption, setSelectedWebhookOption] = useState('none');
  const navigate = useNavigate();

  // Predefined webhook URLs
  const webhookOptions = [
    { id: 'none', label: 'No Webhook', url: '' },
    { id: 'builtin', label: 'Built-in Webhook Receiver', url: 'http://localhost:3004/webhook' },
    { id: 'test', label: 'Test Webhook', url: 'https://webhook.site/your-unique-id' },
    { id: 'custom', label: 'Custom URL', url: '' }
  ];

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

  const handleTakeProfileScreenshot = async () => {
    setIsTakingProfileScreenshot(true);
    try {
      const response = await api.takeProfileScreenshot();
      if (response.status === 'success') {
        setProfileScreenshotUrl(response.screenshot_url);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to take profile screenshot');
    } finally {
      setIsTakingProfileScreenshot(false);
    }
  };

  const handleSubmitTxHash = async () => {
    if (!txHash || !chain) {
      setError('Please provide both transaction hash and chain');
      return;
    }

    setIsSubmitting(true);
    try {
      // If using the mock webhook, handle it locally
      if (selectedWebhookOption === 'builtin') {
        // Create a mock webhook notification
        const mockWebhook: WebhookNotification = {
          status: 'success',
          txHash,
          chain,
          fromAddress: '0x' + Math.random().toString(16).substring(2, 42),
          fromToken: chain === 'ethereum' ? 'ETH' : 'SOL',
          amount: (Math.random() * 10).toFixed(4),
          timestamp: Math.floor(Date.now() / 1000).toString(),
          data: { mock: true }
        };
        
        // Add to local storage
        addWebhook(mockWebhook);
        
        // Update webhook status
        setWebhookStatus(mockWebhook);
        
        // Clear form fields
        setTxHash('');
        setWebhookUrl('');
        
        // Show success message
        setError(null);
        
        // Return early
        setIsSubmitting(false);
        return;
      }
      
      // Otherwise, make the API call
      const response = await api.postTxHash(
        txHash,
        chain,
        selectedWebhookOption !== 'none' ? webhookUrl : undefined
      );
      
      if (response.status === 'success') {
        // Update webhook status if available
        if (response.webhook) {
          setWebhookStatus(response.webhook);
        }
        
        // Clear form fields
        setTxHash('');
        setWebhookUrl('');
        
        // Show success message
        setError(null);
      } else {
        setError(`Transaction submission failed: ${response.status}`);
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to submit transaction hash';
      setError(errorMessage);
      
      // If webhook error, show specific message
      if (errorMessage.includes('webhook') || errorMessage.includes('404')) {
        setError(`Transaction processed but webhook failed: ${errorMessage}. The webhook URL may be invalid or not accessible.`);
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleWebhookOptionChange = (optionId: string) => {
    setSelectedWebhookOption(optionId);
    const selectedOption = webhookOptions.find(option => option.id === optionId);
    
    if (selectedOption) {
      if (optionId === 'custom') {
        setUseWebhook(true);
        setWebhookUrl('');
      } else {
        setUseWebhook(optionId !== 'none');
        setWebhookUrl(selectedOption.url);
      }
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
      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%', mb: 3 }}>
        <UserProfile />
      </Paper>

      <Paper elevation={3} sx={{ p: 4, maxWidth: 600, width: '100%', mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Submit Transaction Hash
        </Typography>
        <Box sx={{ mb: 2 }}>
          <TextField
            fullWidth
            label="Transaction Hash"
            value={txHash}
            onChange={(e) => setTxHash(e.target.value)}
            margin="normal"
          />
          <TextField
            fullWidth
            label="Chain"
            value={chain}
            onChange={(e) => setChain(e.target.value)}
            margin="normal"
          />
          <Typography variant="subtitle1" sx={{ mt: 2, mb: 1 }}>
            Webhook Options
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
            Note: Make sure the webhook URL is accessible before submitting. For testing, you can use the built-in webhook receiver below.
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {webhookOptions.map((option) => (
              <FormControlLabel
                key={option.id}
                control={
                  <Switch
                    checked={selectedWebhookOption === option.id}
                    onChange={() => handleWebhookOptionChange(option.id)}
                  />
                }
                label={option.label}
              />
            ))}
          </Box>
          {selectedWebhookOption === 'custom' && (
            <TextField
              fullWidth
              label="Custom Webhook URL"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              margin="normal"
            />
          )}
          <Button
            variant="contained"
            onClick={handleSubmitTxHash}
            disabled={isSubmitting}
            fullWidth
            sx={{ mt: 2 }}
          >
            {isSubmitting ? 'Submitting...' : 'Submit Transaction Hash'}
          </Button>
        </Box>
        {webhookStatus && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="subtitle1" gutterBottom>
              Webhook Status:
            </Typography>
            <Typography variant="body2">
              Status: {webhookStatus.status}
            </Typography>
            <Typography variant="body2">
              From: {webhookStatus.fromAddress}
            </Typography>
            <Typography variant="body2">
              Amount: {webhookStatus.amount} {webhookStatus.fromToken}
            </Typography>
            <Typography variant="body2">
              Timestamp: {new Date(parseInt(webhookStatus.timestamp) * 1000).toLocaleString()}
            </Typography>
          </Box>
        )}
      </Paper>

      {/* Webhook Receiver Component */}
      <WebhookReceiver refreshInterval={2000} />

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

        <Box sx={{ mb: 3 }}>
          <Button
            variant="outlined"
            startIcon={<CameraAltIcon />}
            onClick={handleTakeProfileScreenshot}
            disabled={isTakingProfileScreenshot}
            fullWidth
          >
            {isTakingProfileScreenshot ? 'Taking Profile Screenshot...' : 'Take Profile Screenshot'}
          </Button>
          {profileScreenshotUrl && (
            <Box sx={{ mt: 2 }}>
              <img
                src={`http://localhost:3005${profileScreenshotUrl}`}
                alt="Profile Screenshot"
                style={{ maxWidth: '100%', height: 'auto' }}
              />
            </Box>
          )}
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
                  View on Twitter
                </Link>
              </Box>
              {screenshotUrl && (
                  <Box sx={{ mt: 2 }}>
                    <img
                        src={`http://localhost:3005${screenshotUrl}`}
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
