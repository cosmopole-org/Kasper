
const drawerWidth = 300;
export const getDrawerWidth = () => drawerWidth

const mixpanelToken = ''
export const getMixPanelToken = () => mixpanelToken

const columns = [
  { name: "ID", uid: "id", sortable: true },
  { name: "KEY", uid: "key", sortable: true },
  { name: "TYPE", uid: "type" },
  { name: "DATA TYPE", uid: "dataType", sortable: true },
  { name: "VALUE", uid: "value" },
  { name: "ACTIONS", uid: "actions" },
];

const playerDataCols = [
  { name: "ID", uid: "id", sortable: true },
  { name: "PLAYER NAME", uid: "playerName" },
  { name: "EMAIL", uid: "email", sortable: true },
  { name: "COIN", uid: "coin", sortable: true },
  { name: "GEM", uid: "gem", sortable: true },
  { name: "ENERGY", uid: "energy", sortable: true },
  { name: "ACTIONS", uid: "actions" },
];

const leaderboardCols = [
  { name: "ID", uid: "userId", sortable: true },
  { name: "NAME", uid: "profile.name" },
  { name: "AVATAR", uid: "profile.avatar" },
  { name: "SCORE", uid: "score" },
  { name: "ACTIONS", uid: "actions" },
];

const dataTypeOptions = [
  { name: "Object", uid: "object" },
  { name: "Number", uid: "number" },
  { name: "Array", uid: "array" },
];

const variables = [
  {
    type: 'number',
    key: "version",
    value: 1,
  },
  {
    type: 'number',
    key: "rewardLeague1Player1",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague1Player2",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague1Player3",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague2Player1",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague2Player2",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague2Player3",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague3Player1",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague3Player2",
    value: 0,
  },
  {
    type: 'number',
    key: "rewardLeague3Player3",
    value: 0,
  },
];

export { columns, playerDataCols, leaderboardCols, variables, dataTypeOptions };
