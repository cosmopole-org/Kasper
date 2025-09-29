import * as React from 'react';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import Avatar from '@mui/material/Avatar';
import ImageIcon from '@mui/icons-material/Image';
import WorkIcon from '@mui/icons-material/Work';
import BeachAccessIcon from '@mui/icons-material/BeachAccess';
import DescriptionIcon from '@mui/icons-material/Description';
import { colors } from '../../App';
import { Typography } from '@mui/material';

export default function ObjectStore() {
  return (
    <List sx={{ width: '100%', maxWidth: 360, bgcolor: 'transparent' }} dense>
      <ListItem>
        <ListItemAvatar>
          <Avatar style={{backgroundColor: colors.colorAccent}}>
            <DescriptionIcon />
          </Avatar>
        </ListItemAvatar>
        <ListItemText primary="Fetch Users" secondary={<Typography style={{color: '#fff'}}>SQL Script</Typography>} style={{color: '#fff'}}/>
      </ListItem>
      <ListItem>
        <ListItemAvatar>
          <Avatar style={{backgroundColor: colors.colorAccent}}>
            <DescriptionIcon />
          </Avatar>
        </ListItemAvatar>
        <ListItemText primary="Authorize User" secondary={<Typography style={{color: '#fff'}}>SQL Script</Typography>} style={{color: '#fff'}}/>
      </ListItem>
      <ListItem>
        <ListItemAvatar>
          <Avatar style={{backgroundColor: colors.colorAccent}}>
            <DescriptionIcon />
          </Avatar>
        </ListItemAvatar>
        <ListItemText primary="Create Account" secondary={<Typography style={{color: '#fff'}}>SQL Script</Typography>} style={{color: '#fff'}}/>
      </ListItem>
    </List>
  );
}
