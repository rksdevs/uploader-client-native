/**
 * Synced from `log-parser-v2/client/lib/raid-data.ts` — boss list order in UI per raid.
 */

export const CATACLYSM_PROGRESSION_SERVERS = new Set(["Whitemane_Gilneas"]);

export function isCataclysmProgressionServer(serverName: string | null | undefined): boolean {
  if (!serverName) return false;
  return CATACLYSM_PROGRESSION_SERVERS.has(serverName);
}

export const raidBossOrder: Record<string, string[]> = {
  "Icecrown Citadel": [
    "Lord Marrowgar",
    "Lady Deathwhisper",
    "Gunship Battle",
    "Deathbringer Saurfang",
    "Festergut",
    "Rotface",
    "Professor Putricide",
    "Blood Prince Council",
    "Blood-Queen Lana'thel",
    "Valithria Dreamwalker",
    "Sindragosa",
    "The Lich King",
  ],
  "Vault of Archavon": [
    "Archavon the Stone Watcher",
    "Emalon the Storm Watcher",
    "Koralon the Flame Watcher",
    "Toravon the Ice Watcher",
  ],
  "The Obsidian Sanctum": ["Sartharion", "Tenebron", "Shadron", "Vesperon"],
  "The Ruby Sanctum": [
    "General Zarithrian",
    "Saviana Ragefire",
    "Baltharus the Warborn",
    "Halion",
  ],
  "Trial of the Crusader": [
    "Northrend Beasts",
    "Lord Jaraxxus",
    "Faction Champions",
    "Twin Val'kyr",
    "Anub'arak",
  ],
  Naxxramas: [
    "Anub'Rekhan",
    "Grand Widow Faerlina",
    "Maexxna",
    "Noth the Plaguebringer",
    "Heigan the Unclean",
    "Loatheb",
    "Instructor Razuvious",
    "Gothik the Harvester",
    "The Four Horsemen",
    "Patchwerk",
    "Grobbulus",
    "Gluth",
    "Thaddius",
    "Sapphiron",
    "Kel'Thuzad",
  ],
  Ulduar: [
    "Flame Leviathan",
    "Ignis the Furnace Master",
    "Razorscale",
    "XT-002 Deconstructor",
    "Assembly of Iron",
    "Kologarn",
    "Auriaya",
    "Hodir",
    "Thorim",
    "Freya",
    "Mimiron",
    "General Vezax",
    "Yogg-Saron",
    "Algalon the Observer",
  ],
  "The Eye of Eternity": ["Malygos"],
  "Onyxia's Lair": ["Onyxia"],
};

export const raidBossOrderCataclysm: Record<string, string[]> = {
  "Blackwing Descent": [
    "Magmaw",
    "Omnotron Defense System",
    "Maloriak",
    "Atramedes",
    "Chimaeron",
    "Nefarian",
  ],
  "The Bastion of Twilight": [
    "Halfus Wyrmbreaker",
    "Valiona and Theralion",
    "Ascendant Council",
    "Cho'gall",
    "Sinestra",
  ],
  "Throne of the Four Winds": ["Conclave of Wind", "Al'Akir"],
  Firelands: [
    "Shannox",
    "Lord Rhyolith",
    "Beth'tilac",
    "Baleroc, the Gatekeeper",
    "Alysrazor",
    "Majordomo Staghelm",
    "Ragnaros",
  ],
  "Baradin Hold": ["Argaloth", "Occu'thar", "Alizabal"],
  "Dragon Soul": [
    "Morchok",
    "Warlord Zon'ozz",
    "Yor'sahj the Unsleeping",
    "Hagara the Stormbinder",
    "Ultraxion",
    "Warmaster Blackhorn",
    "Spine of Deathwing",
    "Madness of Deathwing",
  ],
};

export function raidBossOrderForServer(
  serverName: string | null | undefined
): Record<string, string[]> {
  return isCataclysmProgressionServer(serverName) ? raidBossOrderCataclysm : raidBossOrder;
}
