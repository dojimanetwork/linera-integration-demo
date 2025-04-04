import React, { useEffect, useState } from 'react';
import { Box, Typography, Paper, CircularProgress, Avatar } from '@mui/material';
import VerifiedIcon from '@mui/icons-material/Verified';
import api, { UserDetails } from '../services/api';

const UserProfile: React.FC = () => {
  const [userDetails, setUserDetails] = useState<UserDetails | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [bannerError, setBannerError] = useState(false);

  useEffect(() => {
    const fetchUserDetails = async () => {
      try {
        const response = await api.getUserDetails();
        setUserDetails(response.data);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch user details');
      } finally {
        setLoading(false);
      }
    };

    fetchUserDetails();
  }, []);

  const handleBannerError = () => {
    setBannerError(true);
  };

  // Format banner URL to include dimensions
  const getFormattedBannerUrl = (url: string | null) => {
    if (!url) return null;
    
    // Check if URL already has dimensions
    if (url.includes('/1500x500')) {
      return url;
    }
    
    // Add dimensions to URL
    return `${url}/1500x500`;
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Typography color="error" p={4}>
        Error: {error}
      </Typography>
    );
  }

  if (!userDetails) {
    return (
      <Typography p={4}>
        No user details available
      </Typography>
    );
  }

  const formattedBannerUrl = getFormattedBannerUrl(userDetails.banner_url);

  return (
    <Box>
      {/* Banner Image */}
      {formattedBannerUrl && !bannerError && (
        <Box 
          sx={{ 
            height: 200, 
            width: '100%', 
            borderRadius: '8px 8px 0 0',
            overflow: 'hidden',
            position: 'relative',
            backgroundColor: '#f0f2f5' // Light gray background as fallback
          }}
        >
          <Box
            component="img"
            src={formattedBannerUrl}
            alt="Profile Banner"
            onError={handleBannerError}
            sx={{
              width: '100%',
              height: '100%',
              objectFit: 'cover'
            }}
          />
        </Box>
      )}
      
      {/* Profile Content */}
      <Box sx={{ p: 3, position: 'relative' }}>
        {/* Profile Image */}
        <Box 
          sx={{ 
            position: 'absolute',
            top: formattedBannerUrl && !bannerError ? -50 : 0,
            left: 24,
            zIndex: 1
          }}
        >
          <Avatar
            src={userDetails.profile_image_url}
            alt="Profile"
            sx={{ 
              width: 100, 
              height: 100, 
              border: '4px solid white',
              boxShadow: 2
            }}
          />
        </Box>

        <Box sx={{ mt: formattedBannerUrl && !bannerError ? 8 : 2 }}>
          <Box display="flex" alignItems="center" gap={1}>
            <Typography variant="h5" fontWeight="bold">
              {userDetails.name}
            </Typography>
            {userDetails.verified && (
              <VerifiedIcon color="primary" />
            )}
          </Box>
          
          <Typography color="text.secondary">
            @{userDetails.username}
          </Typography>
          
          <Typography sx={{ mt: 2 }}>
            {userDetails.description}
          </Typography>

          <Box sx={{ mt: 2, display: 'flex', gap: 3 }}>
            <Box>
              <Typography component="span" fontWeight="bold">
                {userDetails.following_count}
              </Typography>
              <Typography component="span" color="text.secondary" sx={{ ml: 0.5 }}>
                Following
              </Typography>
            </Box>
            <Box>
              <Typography component="span" fontWeight="bold">
                {userDetails.followers_count}
              </Typography>
              <Typography component="span" color="text.secondary" sx={{ ml: 0.5 }}>
                Followers
              </Typography>
            </Box>
            <Box>
              <Typography component="span" fontWeight="bold">
                {userDetails.tweet_count}
              </Typography>
              <Typography component="span" color="text.secondary" sx={{ ml: 0.5 }}>
                Tweets
              </Typography>
            </Box>
          </Box>

          <Typography color="text.secondary" sx={{ mt: 2, fontSize: '0.875rem' }}>
            Joined {new Date(userDetails.created_at).toLocaleDateString('en-US', {
              month: 'long',
              year: 'numeric'
            })}
          </Typography>
        </Box>
      </Box>
    </Box>
  );
};

export default UserProfile; 