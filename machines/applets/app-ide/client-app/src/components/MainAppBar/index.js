import * as React from "react";
import { styled, alpha } from "@mui/material/styles";
import AppBar from "@mui/material/AppBar";
import Box from "@mui/material/Box";
import Toolbar from "@mui/material/Toolbar";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import InputBase from "@mui/material/InputBase";
import MenuIcon from "@mui/icons-material/Menu";
import SearchIcon from "@mui/icons-material/Search";
import { colors, currentItemId, forceUpdateApp, setCurrentItemId } from "../../App";
import { Tab, Tabs } from "@mui/material";
import { ThemeProvider } from "@emotion/react";
import { createTheme } from "@mui/material/styles";
import RunIcon from "@mui/icons-material/PlayArrow";
import PauseIcon from "@mui/icons-material/Pause";
import StopIcon from "@mui/icons-material/Stop";
import CloseIcon from '@mui/icons-material/Close';
import { deletePoolItem, pool } from "../../memory";

const tabTheme = createTheme({
  palette: {
    primary: {
      light: "#fff",
      main: "#fff",
      dark: "#fff",
      contrastText: "#fff",
    },
    secondary: {
      light: "#fff",
      main: "#fff",
      dark: "#fff",
      contrastText: "#fff",
    },
  },
});

const Search = styled("div")(({ theme }) => ({
  position: "relative",
  borderRadius: theme.shape.borderRadius,
  backgroundColor: alpha(theme.palette.common.white, 0.15),
  "&:hover": {
    backgroundColor: alpha(theme.palette.common.white, 0.25),
  },
  marginLeft: 0,
  width: "100%",
  [theme.breakpoints.up("sm")]: {
    marginLeft: theme.spacing(1),
    width: "auto",
  },
}));

const SearchIconWrapper = styled("div")(({ theme }) => ({
  padding: theme.spacing(0, 2),
  height: "100%",
  position: "absolute",
  pointerEvents: "none",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
}));

const StyledInputBase = styled(InputBase)(({ theme }) => ({
  color: "inherit",
  "& .MuiInputBase-input": {
    padding: theme.spacing(1, 1, 1, 0),
    // vertical padding + font size from searchIcon
    paddingLeft: `calc(1em + ${theme.spacing(4)})`,
    transition: theme.transitions.create("width"),
    width: "100%",
    [theme.breakpoints.up("sm")]: {
      width: "12ch",
      "&:focus": {
        width: "20ch",
      },
    },
  },
}));

export default function MainAppBar({onDrawerBtnClicked}) {
  const handleChange = (event, newValue) => {
    setCurrentItemId(newValue);
  };
  return (
    <Box sx={{ flexGrow: 1 }}>
      <AppBar
        position="static"
        style={{
          backgroundColor: colors.colorDark2,
          borderRadius: "0px 0px 16px 16px",
        }}
      >
        <Toolbar>
          <IconButton
            size="large"
            edge="start"
            color="inherit"
            aria-label="open drawer"
            sx={{ mr: 2 }}
            onClick={onDrawerBtnClicked}
          >
            <MenuIcon />
          </IconButton>
          <Typography
            variant="h6"
            noWrap
            component="div"
            sx={{ display: { xs: "none", sm: "block" } }}
          >
            Elpis IDE
          </Typography>
          <ThemeProvider theme={tabTheme}>
            <Tabs
              value={currentItemId}
              onChange={handleChange}
              textColor="secondary"
              indicatorColor="secondary"
              centered
              sx={{ flexGrow: 1, display: { xs: "none", sm: "block" } }}
            >
              {Object.keys(pool).map((path) => {
                return (
                  <Tab
                    value={path}
                    label={<><div style={{display: 'flex'}}>{path}<CloseIcon style={{fill: '#fff', marginLeft: 8}} onClick={(e) => {e.preventDefault(); e.stopPropagation(); deletePoolItem(path);}}/></div></>}
                    style={{ color: currentItemId === path ? "#fff" : "#ddd" }}
                  />
                );
              })}
            </Tabs>
          </ThemeProvider>

          <Search style={{ marginRight: 16 }}>
            <SearchIconWrapper>
              <SearchIcon />
            </SearchIconWrapper>
            <StyledInputBase
              placeholder="Search…"
              inputProps={{ "aria-label": "search" }}
            />
          </Search>

          <div
            style={{
              height: 64,
              width: 224,
              marginRight: -24,
              background: "linear-gradient(to right, #1c2f2f, #2a4747)",
              borderRadius: "0px 0px 16px 0px",
              paddingTop: 12,
            }}
          >
            <IconButton
              style={{
                marginLeft: 33,
              }}
              onClick={() => {
                const options = {
                  method: "post",
                  headers: {
                    Accept: "application/json, text/plain, */*",
                    "Content-Type": "application/json",
                  },
                };
                fetch("../run-program", options).then((res) => {});
              }}
            >
              <RunIcon style={{fill: '#fff'}}/>
            </IconButton>
            <IconButton
              style={{
                marginLeft: 24,
              }}
              onClick={() => {
                const options = {
                  method: "post",
                  headers: {
                    Accept: "application/json, text/plain, */*",
                    "Content-Type": "application/json",
                  },
                };
                fetch("../kill-current", options).then((res) => {});
              }}
            >
              <PauseIcon style={{fill: '#fff'}}/>
            </IconButton>
            <IconButton
              style={{
                marginLeft: 24,
              }}
              onClick={() => {
                const options = {
                  method: "post",
                  headers: {
                    Accept: "application/json, text/plain, */*",
                    "Content-Type": "application/json",
                  },
                };
                fetch("../kill-current", options).then((res) => {});
              }}
            >
              <StopIcon style={{fill: '#fff'}}/>
            </IconButton>
          </div>
        </Toolbar>
      </AppBar>
    </Box>
  );
}
